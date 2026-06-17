package main

import (
	"log"
	"os"

	"dungeon-shop/internal/game"
	"dungeon-shop/internal/handlers"
	"dungeon-shop/internal/storage"
	"dungeon-shop/internal/websocket"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{"*"},
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	_, err := storage.NewStorage(redisAddr, mongoURI)
	if err != nil {
		log.Printf("Warning: Failed to connect to storage: %v", err)
		log.Println("Continuing without persistent storage...")
	}

	roomManager := game.NewRoomManager()
	hub := websocket.NewHub(roomManager)
	handler := handlers.NewHandler(roomManager, hub)

	api := e.Group("/api")
	api.POST("/rooms", handler.CreateRoom)
	api.GET("/rooms", handler.ListRooms)
	api.GET("/rooms/:roomId", handler.GetRoom)
	api.POST("/rooms/:roomId/join", handler.JoinRoom)
	api.POST("/rooms/:roomId/start", handler.StartGame)
	api.GET("/item-types", handler.GetItemTypes)

	e.GET("/ws/:roomId", handler.WSHandler)

	log.Printf("Server starting on port %s...", serverPort)
	if err := e.Start(":" + serverPort); err != nil {
		log.Fatal(err)
	}
}
