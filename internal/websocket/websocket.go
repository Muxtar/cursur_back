package websocket

import (
	"chat-backend/internal/database"
	"chat-backend/internal/utils"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func HandleWebSocket(hub *Hub, c *gin.Context, db *database.Database) {
	// Get user ID from token
	token := c.Query("token")
	if token == "" {
		c.JSON(401, gin.H{"error": "Token required"})
		return
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:    claims.UserID,
		Conn:  conn,
		Hub:   hub,
		Send:  make(chan []byte, 256),
		Chats: make(map[primitive.ObjectID]bool),
	}

	client.Hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle different message types
		switch msg["type"] {
		case "join_chat":
			if chatIDStr, ok := msg["chat_id"].(string); ok {
				chatID, _ := primitive.ObjectIDFromHex(chatIDStr)
				c.Chats[chatID] = true
			}
		case "leave_chat":
			if chatIDStr, ok := msg["chat_id"].(string); ok {
				chatID, _ := primitive.ObjectIDFromHex(chatIDStr)
				delete(c.Chats, chatID)
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}

