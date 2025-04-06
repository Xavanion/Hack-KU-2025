package main

import (
	"net/http"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	roomhandler "github.com/Xavanion/Hack-KU-2025/backend/room-handling"
)

// Upgrade HTTP request to WS connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	// Create a new manager to handler all rooms that get spun off later
	roomManager := roomhandler.NewRoomManager()
	// default room for now
	roomManager.CreateRoom("one")

	router := gin.Default()
	// Serve frontend static files
	router.Use(static.Serve("/", static.LocalFile("real-time-app/dist", true)))

	// Setup route group for the API
	api := router.Group("/api")
	{
		// API pong response
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Websocket endpoint
		api.GET("/ws", func(c *gin.Context) {
			// Upgrade GET request to a WebSocket
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				return
			}
			// close connection when we're done
			defer conn.Close()
			roomManager.GetRoom("one").NewConnection(conn)
		})
	}
	router.Run(":8080")
}
