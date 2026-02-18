package websocket

import (
	"context"
	"chat-backend/internal/database"
	"chat-backend/internal/models"
	"chat-backend/internal/utils"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production (Railway), we should check origin
		// But for now, allow all origins to ensure WebSocket works
		// TODO: Add proper origin checking based on CORS_ALLOWED_ORIGINS
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Some clients don't send Origin header, allow them
			return true
		}
		// Allow all origins for now (can be restricted later)
		return true
	},
	// Enable compression for better performance
	EnableCompression: true,
}

func HandleWebSocket(hub *Hub, c *gin.Context, db *database.Database) {
	// Log WebSocket connection attempt
	origin := c.Request.Header.Get("Origin")
	log.Printf("🔌 WebSocket connection attempt from: %s, Origin: %s", c.Request.RemoteAddr, origin)
	
	// Get user ID from token
	token := c.Query("token")
	if token == "" {
		log.Printf("❌ WebSocket connection rejected: Token required")
		c.JSON(401, gin.H{"error": "Token required"})
		return
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		log.Printf("❌ WebSocket connection rejected: Invalid token - %v", err)
		c.JSON(401, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		return
	}
	
	log.Printf("✅ WebSocket connection established for user: %s", claims.UserID.Hex())

	client := &Client{
		ID:    claims.UserID,
		Conn:  conn,
		Hub:   hub,
		Send:  make(chan []byte, 256),
		Chats: make(map[primitive.ObjectID]bool),
		DB:    db,
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
				// Subscribe client to room in the hub (so room broadcasts actually work)
				c.Hub.joinRoom <- roomEvent{client: c, chatID: chatID}
			}
		case "leave_chat":
			if chatIDStr, ok := msg["chat_id"].(string); ok {
				chatID, _ := primitive.ObjectIDFromHex(chatIDStr)
				// Unsubscribe client from room
				c.Hub.leaveRoom <- roomEvent{client: c, chatID: chatID}
			}
		case "webrtc_offer", "webrtc_answer", "webrtc_ice":
			// Forward WebRTC signaling messages to call members (not chat members)
			// This ensures only participants of the specific call receive WebRTC signals
			if chatIDStr, ok := msg["chat_id"].(string); ok {
				chatID, err := primitive.ObjectIDFromHex(chatIDStr)
				if err == nil {
					msgBytes, _ := json.Marshal(msg)
					
					// Try to get call_id from message
					callIDStr, hasCallID := msg["call_id"].(string)
					var callID primitive.ObjectID
					if hasCallID {
						callID, _ = primitive.ObjectIDFromHex(callIDStr)
					}
					
					// Forward to room excluding sender (so sender doesn't receive their own WebRTC messages)
					// This helps when users are actively in the chat room
					c.Hub.BroadcastJSONToRoomExcludingSender(chatID, c.ID, msgBytes)
					
					// Send directly to call members (not chat members)
					// This ensures WebRTC signaling works even if the callee hasn't opened the chat UI
					if c.DB != nil && c.DB.MongoDB != nil {
						// If we have call_id, use call members; otherwise fallback to chat members
						if hasCallID && !callID.IsZero() {
							var call models.Call
							if err := c.DB.MongoDB.Collection("calls").FindOne(
								context.Background(),
								bson.M{"_id": callID},
							).Decode(&call); err == nil {
								// Send to call members only
								for _, memberID := range call.Members {
									if memberID == c.ID {
										continue // Skip sender
									}
									c.Hub.SendJSONToUser(memberID, msgBytes)
								}
							} else {
								// Call not found, fallback to chat members
								var chat models.Chat
								if err := c.DB.MongoDB.Collection("chats").FindOne(
									context.Background(),
									bson.M{"_id": chatID},
								).Decode(&chat); err == nil {
									for _, memberID := range chat.Members {
										if memberID == c.ID {
											continue
										}
										c.Hub.SendJSONToUser(memberID, msgBytes)
									}
								}
							}
						} else {
							// No call_id, fallback to chat members (backward compatibility)
							var chat models.Chat
							if err := c.DB.MongoDB.Collection("chats").FindOne(
								context.Background(),
								bson.M{"_id": chatID},
							).Decode(&chat); err == nil {
								for _, memberID := range chat.Members {
									if memberID == c.ID {
										continue
									}
									c.Hub.SendJSONToUser(memberID, msgBytes)
								}
							}
						}
					}
				}
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

