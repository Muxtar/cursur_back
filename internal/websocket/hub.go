package websocket

import (
	"chat-backend/internal/models"
	"encoding/json"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	ID     primitive.ObjectID
	Conn   *websocket.Conn
	Hub    *Hub
	Send   chan []byte
	Chats  map[primitive.ObjectID]bool
}

type roomSubscription struct {
	ChatID primitive.ObjectID
	Client *Client
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	rooms      map[primitive.ObjectID]map[*Client]bool
	// Track online users by user ID
	onlineUsers map[primitive.ObjectID]bool

	joinRoom  chan roomSubscription
	leaveRoom chan roomSubscription
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister: make(chan *Client),
		rooms:       make(map[primitive.ObjectID]map[*Client]bool),
		onlineUsers: make(map[primitive.ObjectID]bool),
		joinRoom:    make(chan roomSubscription),
		leaveRoom:   make(chan roomSubscription),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.onlineUsers[client.ID] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				// Check if user has any other active connections
				hasOtherConnection := false
				for c := range h.clients {
					if c.ID == client.ID {
						hasOtherConnection = true
						break
					}
				}
				if !hasOtherConnection {
					delete(h.onlineUsers, client.ID)
				}
				// Remove client from any rooms it joined
				for chatID, room := range h.rooms {
					if _, ok := room[client]; ok {
						delete(room, client)
						if len(room) == 0 {
							delete(h.rooms, chatID)
						}
					}
				}
			}

		case sub := <-h.joinRoom:
			if h.rooms[sub.ChatID] == nil {
				h.rooms[sub.ChatID] = make(map[*Client]bool)
			}
			h.rooms[sub.ChatID][sub.Client] = true

		case sub := <-h.leaveRoom:
			if room, ok := h.rooms[sub.ChatID]; ok {
				delete(room, sub.Client)
				if len(room) == 0 {
					delete(h.rooms, sub.ChatID)
				}
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(chatID primitive.ObjectID, message models.Message) {
	// Front-end expects JSON "message" events (at minimum chat_id)
	event := map[string]interface{}{
		"type":       "message",
		"chat_id":    chatID.Hex(),
		"message_id": message.ID.Hex(),
		"sender_id":  message.SenderID.Hex(),
		"status":     message.Status,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.BroadcastJSONToRoom(chatID, data)
}

// BroadcastJSONToRoom broadcasts a JSON message to all clients in a chat room
func (h *Hub) BroadcastJSONToRoom(chatID primitive.ObjectID, data []byte) {
	if room, ok := h.rooms[chatID]; ok {
		for client := range room {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastJSONToUsers broadcasts a JSON message to all active connections of given user IDs.
func (h *Hub) BroadcastJSONToUsers(userIDs []primitive.ObjectID, data []byte) {
	if len(userIDs) == 0 {
		return
	}
	targets := make(map[primitive.ObjectID]struct{}, len(userIDs))
	for _, id := range userIDs {
		targets[id] = struct{}{}
	}
	for client := range h.clients {
		if _, ok := targets[client.ID]; !ok {
			continue
		}
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.clients, client)
		}
	}
}

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID primitive.ObjectID) bool {
	return h.onlineUsers[userID]
}

// GetOnlineUsers returns list of online user IDs
func (h *Hub) GetOnlineUsers() []primitive.ObjectID {
	online := make([]primitive.ObjectID, 0, len(h.onlineUsers))
	for userID := range h.onlineUsers {
		online = append(online, userID)
	}
	return online
}





