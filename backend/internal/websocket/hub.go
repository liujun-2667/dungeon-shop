package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"dungeon-shop/internal/game"
	"dungeon-shop/internal/models"

	"github.com/gorilla/websocket"
)

type Hub struct {
	roomManager *game.RoomManager
	clients     map[string]map[*Client]bool
	broadcast   chan models.WSMessage
	Register    chan *Client
	Unregister  chan *Client
	mu          sync.RWMutex
}

type Client struct {
	Conn     *websocket.Conn
	Send     chan models.WSMessage
	RoomID   string
	PlayerID string
}

func NewHub(roomManager *game.RoomManager) *Hub {
	hub := &Hub{
		roomManager: roomManager,
		clients:     make(map[string]map[*Client]bool),
		broadcast:   make(chan models.WSMessage),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
	}
	go hub.run()
	go hub.phaseTimer()
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.clients[client.RoomID] == nil {
				h.clients[client.RoomID] = make(map[*Client]bool)
			}
			h.clients[client.RoomID][client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.RoomID]; ok {
				delete(h.clients[client.RoomID], client)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.clients[message.RoomID]; ok {
				for client := range clients {
					select {
					case client.Send <- message:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) phaseTimer() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		rooms := h.roomManager.ListRooms()
		now := time.Now().Unix()

		for _, room := range rooms {
			if room.Status != "playing" {
				continue
			}

			if room.Phase == models.PhaseBusiness {
				pending, targetPlayerID := h.roomManager.HasPendingBargain(room.ID)
				if pending != nil && now >= pending.ExpiresAt {
					bargainID := pending.ID
					targetPlayerIDLocal := targetPlayerID
					logs := h.roomManager.ResolveBargain(room.ID, bargainID, false)
					h.dispatchBusinessLogs(room.ID, logs)

					h.BroadcastToRoom(room.ID, models.WSMessage{
						Type:   "room_update",
						RoomID: room.ID,
						Data:   room,
					})

					h.SendToPlayer(room.ID, targetPlayerIDLocal, models.WSMessage{
						Type:   "bargain_timeout",
						RoomID: room.ID,
						Data:   bargainID,
					})
				}

				if pending == nil {
					for i := 0; i < 3; i++ {
						bargain, bargainTarget, logs, hasMore := h.roomManager.ProcessNextNPC(room.ID)
						h.dispatchBusinessLogs(room.ID, logs)

						if bargain != nil {
							bargainTargetLocal := bargainTarget
							h.BroadcastToRoom(room.ID, models.WSMessage{
								Type:   "room_update",
								RoomID: room.ID,
								Data:   room,
							})

							h.SendToPlayer(room.ID, bargainTargetLocal, models.WSMessage{
								Type:   "bargain_request",
								RoomID: room.ID,
								Data:   bargain,
							})
							break
						}

						if len(logs) > 0 || hasMore {
							h.BroadcastToRoom(room.ID, models.WSMessage{
								Type:   "room_update",
								RoomID: room.ID,
								Data:   room,
							})
						}

						if !hasMore {
							break
						}
					}
				}
			}

			if now >= room.PhaseEndTime {
				prevAuctions := make(map[string]string)
				for _, a := range room.Auctions {
					if a.Status == "active" {
						prevAuctions[a.ID] = string(a.Status)
					}
				}

				h.roomManager.ProcessPhaseEnd(room.ID)

				h.BroadcastToRoom(room.ID, models.WSMessage{
					Type:   "phase_update",
					RoomID: room.ID,
					Data: map[string]interface{}{
						"phase":       room.Phase,
						"week":        room.CurrentWeek,
						"phaseEndTime": room.PhaseEndTime,
					},
				})

				for _, a := range room.Auctions {
					if prevAuctions[a.ID] == "active" && a.Status != "active" {
						auctionEndMsg := models.WSMessage{
							Type:   "auction_end",
							RoomID: room.ID,
							Data: map[string]interface{}{
								"auctionId":     a.ID,
								"status":        string(a.Status),
								"currentPrice":  a.CurrentPrice,
								"highestBidder": a.HighestBidderName,
								"itemTypeName":  a.ItemTypeName,
							},
						}
						if a.IsGuildAuction {
							h.SendToGuild(room.ID, a.GuildID, auctionEndMsg)
						} else {
							h.BroadcastToRoom(room.ID, auctionEndMsg)
						}
					}
				}

				h.BroadcastToRoom(room.ID, models.WSMessage{
					Type:   "room_update",
					RoomID: room.ID,
					Data:   room,
				})

				h.broadcastReputationUpdates(room.ID, room)

				if room.Status == "finished" {
					results := h.calculateResults(room)
					h.BroadcastToRoom(room.ID, models.WSMessage{
						Type:   "game_end",
						RoomID: room.ID,
						Data:   results,
					})
				}
			}
		}
	}
}

func (h *Hub) dispatchBusinessLogs(roomID string, logs []models.BusinessLogEntry) {
	if logs == nil || len(logs) == 0 {
		return
	}
	for _, log := range logs {
		msg := log.NPCName + ": " + log.Message
		h.BroadcastToRoom(roomID, models.WSMessage{
			Type:     "business_log",
			RoomID:   roomID,
			PlayerID: log.PlayerID,
			Data: map[string]interface{}{
				"message": msg,
				"type":    log.Type,
				"npcName": log.NPCName,
			},
		})
	}
}

func (h *Hub) calculateResults(room *models.Room) []models.GamePlayerResult {
	results := make([]models.GamePlayerResult, 0, len(room.Players))

	for _, player := range room.Players {
		assets := game.CalculateTotalAssets(player)
		results = append(results, models.GamePlayerResult{
			PlayerID:    player.ID,
			Name:        player.Name,
			FinalAssets: assets,
		})
	}

	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[j].FinalAssets > results[i].FinalAssets {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	for i := range results {
		results[i].Rank = i + 1
		results[i].IsWinner = i == 0
	}

	return results
}

func (h *Hub) BroadcastToRoom(roomID string, message models.WSMessage) {
	h.broadcast <- message
}

func (h *Hub) SendToPlayer(roomID, playerID string, message models.WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[roomID]; ok {
		for client := range clients {
			if client.PlayerID == playerID {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(clients, client)
				}
				return
			}
		}
	}
}

func (h *Hub) SendToGuild(roomID, guildID string, message models.WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.roomManager.GetRoom(roomID)
	if !ok {
		return
	}

	guild, ok := room.Guilds[guildID]
	if !ok {
		return
	}

	guildMemberIDs := make(map[string]bool)
	for _, member := range guild.Members {
		guildMemberIDs[member.PlayerID] = true
	}

	if clients, ok := h.clients[roomID]; ok {
		for client := range clients {
			if guildMemberIDs[client.PlayerID] {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(clients, client)
				}
			}
		}
	}
}

func (h *Hub) broadcastGuildUpdate(roomID string) {
	room, ok := h.roomManager.GetRoom(roomID)
	if !ok {
		return
	}

	guildsList := make([]*models.Guild, 0, len(room.Guilds))
	for _, guild := range room.Guilds {
		guildsList = append(guildsList, guild)
	}

	guildRankings := make(map[string][]models.GuildMemberRank)
	for _, guild := range room.Guilds {
		ranking := h.roomManager.GetGuildMemberRanking(roomID, guild.ID)
		guildRankings[guild.ID] = ranking
	}

	h.BroadcastToRoom(roomID, models.WSMessage{
		Type:   "guild_update",
		RoomID: roomID,
		Data: map[string]interface{}{
			"guilds":   guildsList,
			"rankings": guildRankings,
		},
	})

	for _, guild := range room.Guilds {
		h.sendGuildInternalUpdate(roomID, guild.ID, guild, guildRankings[guild.ID])
	}
}

func (h *Hub) sendGuildInternalUpdate(roomID, guildID string, guild *models.Guild, ranking []models.GuildMemberRank) {
	if guild == nil {
		room, ok := h.roomManager.GetRoom(roomID)
		if !ok {
			return
		}
		var gok bool
		guild, gok = room.Guilds[guildID]
		if !gok {
			return
		}
		if ranking == nil {
			ranking = h.roomManager.GetGuildMemberRanking(roomID, guildID)
		}
	}

	h.SendToGuild(roomID, guildID, models.WSMessage{
		Type:   "guild_internal_update",
		RoomID: roomID,
		Data: map[string]interface{}{
			"guildId":  guildID,
			"tasks":    guild.Tasks,
			"logs":     guild.Logs,
			"ranking":  ranking,
		},
	})
}

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("error parsing message: %v", err)
			continue
		}

		msg.RoomID = c.RoomID
		msg.PlayerID = c.PlayerID

		hub.handleMessage(msg)
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Printf("error marshaling message: %v", err)
			return
		}

		if err := c.Conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			log.Printf("error writing message: %v", err)
			return
		}
	}
}

func (h *Hub) handleMessage(msg models.WSMessage) {
	room, ok := h.roomManager.GetRoom(msg.RoomID)
	if !ok {
		return
	}

	switch msg.Type {
	case "buy_item":
		var data struct {
			ItemTypeID string          `json:"itemTypeId"`
			Quality    models.Quality  `json:"quality"`
			Quantity   int             `json:"quantity"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.BuyFromWholesaler(msg.RoomID, msg.PlayerID, data.ItemTypeID, data.Quality, data.Quantity) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "place_item":
		var data struct {
			ItemID  string `json:"itemId"`
			ShelfID string `json:"shelfId"`
			Price   int    `json:"price"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.PlaceItemOnShelf(msg.RoomID, msg.PlayerID, data.ItemID, data.ShelfID, data.Price) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "set_price":
		var data struct {
			ShelfID string `json:"shelfId"`
			Price   int    `json:"price"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.SetShelfPrice(msg.RoomID, msg.PlayerID, data.ShelfID, data.Price) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "remove_item":
		var data struct {
			ShelfID string `json:"shelfId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.RemoveItemFromShelf(msg.RoomID, msg.PlayerID, data.ShelfID) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "hire_adventurer":
		var data struct {
			AdventurerIdx int `json:"adventurerIdx"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if _, ok := h.roomManager.HireAdventurer(msg.RoomID, msg.PlayerID, data.AdventurerIdx); ok {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "dispatch_adventurer":
		var data struct {
			AdventurerID string `json:"adventurerId"`
			Floor        int    `json:"floor"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.DispatchAdventurer(msg.RoomID, msg.PlayerID, data.AdventurerID, data.Floor) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "start_synthesis":
		var data struct {
			RecipeID string `json:"recipeId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.StartSynthesis(msg.RoomID, msg.PlayerID, data.RecipeID) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "upgrade_shop":
		var data struct {
			UpgradeType string `json:"upgradeType"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if h.roomManager.UpgradeShop(msg.RoomID, msg.PlayerID, data.UpgradeType) {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
		}

	case "chat":
		h.BroadcastToRoom(msg.RoomID, models.WSMessage{
			Type:     "chat",
			RoomID:   msg.RoomID,
			PlayerID: msg.PlayerID,
			Data:     msg.Data,
		})

	case "bargain_accept":
		var data struct {
			BargainID string `json:"bargainId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if data.BargainID != "" {
			logs := h.roomManager.ResolveBargain(msg.RoomID, data.BargainID, true)
			h.dispatchBusinessLogs(msg.RoomID, logs)

			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "bargain_resolved",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"bargainId": data.BargainID, "accepted": true},
			})
		}

	case "bargain_reject":
		var data struct {
			BargainID string `json:"bargainId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		if data.BargainID != "" {
			logs := h.roomManager.ResolveBargain(msg.RoomID, data.BargainID, false)
			h.dispatchBusinessLogs(msg.RoomID, logs)

			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "bargain_resolved",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"bargainId": data.BargainID, "accepted": false},
			})
		}

	case "create_auction":
		var data struct {
			ItemID        string `json:"itemId"`
			StartingPrice int    `json:"startingPrice"`
			BuyoutPrice   int    `json:"buyoutPrice"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		auction, errMsg := h.roomManager.CreateAuction(msg.RoomID, msg.PlayerID, data.ItemID, data.StartingPrice, data.BuyoutPrice)
		if auction != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "auction_created",
				RoomID: msg.RoomID,
				Data:   auction,
			})
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "auction_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "create_auction", "error": errMsg},
			})
		}

	case "place_bid":
		var data struct {
			AuctionID string `json:"auctionId"`
			BidAmount int    `json:"bidAmount"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		deposit := int(float64(data.BidAmount) * 0.10)
		auction, errMsg := h.roomManager.PlaceBid(msg.RoomID, msg.PlayerID, data.AuctionID, data.BidAmount)
		if auction != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			if auction.Status == models.AuctionSold {
				buyoutMsg := models.WSMessage{
					Type:   "buyout",
					RoomID: msg.RoomID,
					Data: map[string]interface{}{
						"auctionId":     auction.ID,
						"currentPrice":  auction.CurrentPrice,
						"highestBidder": auction.HighestBidderName,
						"itemTypeName":  auction.ItemTypeName,
					},
				}
				if auction.IsGuildAuction {
					h.SendToGuild(msg.RoomID, auction.GuildID, buyoutMsg)
				} else {
					h.BroadcastToRoom(msg.RoomID, buyoutMsg)
				}
				h.broadcastReputationUpdates(msg.RoomID, room)
			} else {
				bidMsg := models.WSMessage{
					Type:   "bid_update",
					RoomID: msg.RoomID,
					Data: map[string]interface{}{
						"auctionId":      auction.ID,
						"currentPrice":   auction.CurrentPrice,
						"highestBidder":  auction.HighestBidderName,
						"itemTypeName":   auction.ItemTypeName,
						"remainingWeeks": auction.EndWeek - room.CurrentWeek,
						"deposit":        deposit,
						"bidderId":       msg.PlayerID,
					},
				}
				if auction.IsGuildAuction {
					h.SendToGuild(msg.RoomID, auction.GuildID, bidMsg)
				} else {
					h.BroadcastToRoom(msg.RoomID, bidMsg)
				}
			}
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "auction_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "place_bid", "error": errMsg},
			})
		}

	case "buyout_auction":
		var data struct {
			AuctionID string `json:"auctionId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		auction, errMsg := h.roomManager.Buyout(msg.RoomID, msg.PlayerID, data.AuctionID)
		if auction != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			buyoutMsg := models.WSMessage{
				Type:   "buyout",
				RoomID: msg.RoomID,
				Data: map[string]interface{}{
					"auctionId":     auction.ID,
					"currentPrice":  auction.CurrentPrice,
					"highestBidder": auction.HighestBidderName,
					"itemTypeName":  auction.ItemTypeName,
				},
			}
			if auction.IsGuildAuction {
				h.SendToGuild(msg.RoomID, auction.GuildID, buyoutMsg)
			} else {
				h.BroadcastToRoom(msg.RoomID, buyoutMsg)
			}
			h.broadcastReputationUpdates(msg.RoomID, room)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "auction_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "buyout", "error": errMsg},
			})
		}

	case "cancel_auction":
		var data struct {
			AuctionID string `json:"auctionId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		auction, errMsg := h.roomManager.CancelAuction(msg.RoomID, msg.PlayerID, data.AuctionID)
		if auction != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			if auction.IsGuildAuction {
				h.SendToGuild(msg.RoomID, auction.GuildID, models.WSMessage{
					Type:   "auction_cancelled",
					RoomID: msg.RoomID,
					Data:   auction.ID,
				})
			} else {
				h.BroadcastToRoom(msg.RoomID, models.WSMessage{
					Type:   "auction_cancelled",
					RoomID: msg.RoomID,
					Data:   auction.ID,
				})
			}
			h.broadcastReputationUpdates(msg.RoomID, room)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "auction_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "cancel", "error": errMsg},
			})
		}

	case "create_guild":
		var data struct {
			GuildName string `json:"guildName"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		guild, errMsg := h.roomManager.CreateGuild(msg.RoomID, msg.PlayerID, data.GuildName)
		if guild != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "create_guild", "error": errMsg},
			})
		}

	case "join_guild":
		var data struct {
			GuildID string `json:"guildId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		guild, errMsg := h.roomManager.JoinGuild(msg.RoomID, msg.PlayerID, data.GuildID)
		if guild != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "join_guild", "error": errMsg},
			})
		}

	case "leave_guild":
		errMsg := h.roomManager.LeaveGuild(msg.RoomID, msg.PlayerID)
		if errMsg == "" {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "leave_guild", "error": errMsg},
			})
		}

	case "kick_guild_member":
		var data struct {
			TargetPlayerID string `json:"targetPlayerId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		errMsg := h.roomManager.KickMember(msg.RoomID, msg.PlayerID, data.TargetPlayerID)
		if errMsg == "" {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "kick_member", "error": errMsg},
			})
		}

	case "upgrade_guild":
		guild, errMsg := h.roomManager.UpgradeGuild(msg.RoomID, msg.PlayerID)
		if guild != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "upgrade_guild", "error": errMsg},
			})
		}

	case "deposit_guild_warehouse":
		var data struct {
			ItemID string `json:"itemId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		errMsg := h.roomManager.DepositGuildWarehouse(msg.RoomID, msg.PlayerID, data.ItemID)
		if errMsg == "" {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "deposit_warehouse", "error": errMsg},
			})
		}

	case "withdraw_guild_warehouse":
		var data struct {
			ItemID string `json:"itemId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		errMsg := h.roomManager.WithdrawGuildWarehouse(msg.RoomID, msg.PlayerID, data.ItemID)
		if errMsg == "" {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "withdraw_warehouse", "error": errMsg},
			})
		}

	case "create_guild_auction":
		var data struct {
			ItemID        string `json:"itemId"`
			StartingPrice int    `json:"startingPrice"`
			BuyoutPrice   int    `json:"buyoutPrice"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		auction, errMsg := h.roomManager.CreateGuildAuction(msg.RoomID, msg.PlayerID, data.ItemID, data.StartingPrice, data.BuyoutPrice)
		if auction != nil {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})

			h.SendToGuild(msg.RoomID, auction.GuildID, models.WSMessage{
				Type:   "guild_auction_created",
				RoomID: msg.RoomID,
				Data:   auction,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "auction_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "create_guild_auction", "error": errMsg},
			})
		}

	case "transfer_guild_leadership":
		var data struct {
			TargetPlayerID string `json:"targetPlayerId"`
		}
		dataBytes, _ := json.Marshal(msg.Data)
		json.Unmarshal(dataBytes, &data)

		errMsg := h.roomManager.TransferGuildLeadership(msg.RoomID, msg.PlayerID, data.TargetPlayerID)
		if errMsg == "" {
			h.BroadcastToRoom(msg.RoomID, models.WSMessage{
				Type:   "room_update",
				RoomID: msg.RoomID,
				Data:   room,
			})
			h.broadcastGuildUpdate(msg.RoomID)
		} else {
			h.SendToPlayer(msg.RoomID, msg.PlayerID, models.WSMessage{
				Type:   "guild_error",
				RoomID: msg.RoomID,
				Data:   map[string]interface{}{"action": "transfer_leadership", "error": errMsg},
			})
		}
	}
}

func (h *Hub) broadcastReputationUpdates(roomID string, room *models.Room) {
	reputationData := make([]map[string]interface{}, 0)
	for _, player := range room.Players {
		reputationData = append(reputationData, map[string]interface{}{
			"playerId":           player.ID,
			"auctionReputation":  player.AuctionReputation,
			"shopReputation":     player.Reputation,
		})
	}
	h.BroadcastToRoom(roomID, models.WSMessage{
		Type:   "reputation_update",
		RoomID: roomID,
		Data:   reputationData,
	})
}
