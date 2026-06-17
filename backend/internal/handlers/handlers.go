package handlers

import (
	"net/http"
	"time"

	"dungeon-shop/internal/game"
	"dungeon-shop/internal/models"
	ws "dungeon-shop/internal/websocket"

	gorillaWs "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = gorillaWs.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	roomManager *game.RoomManager
	hub         *ws.Hub
}

func NewHandler(roomManager *game.RoomManager, hub *ws.Hub) *Handler {
	return &Handler{
		roomManager: roomManager,
		hub:         hub,
	}
}

type CreateRoomRequest struct {
	Name       string `json:"name"`
	MaxPlayers int    `json:"maxPlayers"`
	PlayerName string `json:"playerName"`
	ShopName   string `json:"shopName"`
}

type CreateRoomResponse struct {
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
}

func (h *Handler) CreateRoom(c echo.Context) error {
	var req CreateRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.MaxPlayers < 2 || req.MaxPlayers > 4 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Max players must be 2-4"})
	}

	seed := time.Now().UnixNano()
	room := h.roomManager.CreateRoom(req.Name, req.MaxPlayers, seed)

	_, player, ok := h.roomManager.JoinRoom(room.ID, req.PlayerName, req.ShopName)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to join room"})
	}

	return c.JSON(http.StatusOK, CreateRoomResponse{
		RoomID:   room.ID,
		PlayerID: player.ID,
	})
}

type JoinRoomRequest struct {
	PlayerName string `json:"playerName"`
	ShopName   string `json:"shopName"`
}

func (h *Handler) JoinRoom(c echo.Context) error {
	roomID := c.Param("roomId")

	var req JoinRoomRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	_, player, ok := h.roomManager.JoinRoom(roomID, req.PlayerName, req.ShopName)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to join room"})
	}

	h.hub.BroadcastToRoom(roomID, models.WSMessage{
		Type:   "player_joined",
		RoomID: roomID,
		Data:   player,
	})

	return c.JSON(http.StatusOK, CreateRoomResponse{
		RoomID:   roomID,
		PlayerID: player.ID,
	})
}

func (h *Handler) ListRooms(c echo.Context) error {
	rooms := h.roomManager.ListRooms()
	return c.JSON(http.StatusOK, rooms)
}

func (h *Handler) GetRoom(c echo.Context) error {
	roomID := c.Param("roomId")
	room, ok := h.roomManager.GetRoom(roomID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
	}
	return c.JSON(http.StatusOK, room)
}

func (h *Handler) StartGame(c echo.Context) error {
	roomID := c.Param("roomId")

	if !h.roomManager.StartGame(roomID) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to start game"})
	}

	room, _ := h.roomManager.GetRoom(roomID)

	h.hub.BroadcastToRoom(roomID, models.WSMessage{
		Type:   "game_start",
		RoomID: roomID,
		Data:   room,
	})

	return c.JSON(http.StatusOK, map[string]string{"status": "started"})
}

func (h *Handler) GetItemTypes(c echo.Context) error {
	return c.JSON(http.StatusOK, models.ItemTypes)
}

func (h *Handler) WSHandler(c echo.Context) error {
	roomID := c.Param("roomId")
	playerID := c.QueryParam("playerId")

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := &ws.Client{
		Conn:     conn,
		Send:     make(chan models.WSMessage, 256),
		RoomID:   roomID,
		PlayerID: playerID,
	}

	h.hub.Register <- client

	room, _ := h.roomManager.GetRoom(roomID)
	client.Send <- models.WSMessage{
		Type:   "room_state",
		RoomID: roomID,
		Data:   room,
	}

	go client.WritePump()
	go client.ReadPump(h.hub)

	return nil
}
