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
				pending := h.roomManager.HasPendingBargain(room.ID)
				if pending != nil && now >= pending.ExpiresAt {
					targetPlayerID := room.BargainPlayerID
					bargainID := pending.ID
					logs := h.roomManager.ResolveBargain(room.ID, bargainID, false)
					h.dispatchBusinessLogs(room.ID, logs)

					h.BroadcastToRoom(room.ID, models.WSMessage{
						Type:   "room_update",
						RoomID: room.ID,
						Data:   room,
					})

					h.SendToPlayer(room.ID, targetPlayerID, models.WSMessage{
						Type:   "bargain_timeout",
						RoomID: room.ID,
						Data:   bargainID,
					})
				}

				if pending == nil {
					for i := 0; i < 3; i++ {
						bargain, logs, hasMore := h.roomManager.ProcessNextNPC(room.ID)
						h.dispatchBusinessLogs(room.ID, logs)

						if bargain != nil {
							bargainTarget := room.BargainPlayerID
							h.BroadcastToRoom(room.ID, models.WSMessage{
								Type:   "room_update",
								RoomID: room.ID,
								Data:   room,
							})

							h.SendToPlayer(room.ID, bargainTarget, models.WSMessage{
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

				h.BroadcastToRoom(room.ID, models.WSMessage{
					Type:   "room_update",
					RoomID: room.ID,
					Data:   room,
				})

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
	}
}
