package websocket

import (
	"chat-backend/internal/models"
	"sync"

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

type roomEvent struct {
	client *Client
	chatID primitive.ObjectID
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	rooms      map[primitive.ObjectID]map[*Client]bool
	// Track online users by user ID
	onlineUsers map[primitive.ObjectID]bool

	joinRoom  chan roomEvent
	leaveRoom chan roomEvent
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister: make(chan *Client),
		rooms:       make(map[primitive.ObjectID]map[*Client]bool),
		onlineUsers: make(map[primitive.ObjectID]bool),
		joinRoom:    make(chan roomEvent, 64),
		leaveRoom:   make(chan roomEvent, 64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.onlineUsers[client.ID] = true
			for chatID := range client.Chats {
				if h.rooms[chatID] == nil {
					h.rooms[chatID] = make(map[*Client]bool)
				}
				h.rooms[chatID][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
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
				for chatID := range client.Chats {
					if room, ok := h.rooms[chatID]; ok {
						delete(room, client)
						if len(room) == 0 {
							delete(h.rooms, chatID)
						}
					}
				}
			}
			h.mu.Unlock()

		case ev := <-h.joinRoom:
			h.mu.Lock()
			// Keep client-side chat list consistent for cleanup.
			ev.client.Chats[ev.chatID] = true
			if h.rooms[ev.chatID] == nil {
				h.rooms[ev.chatID] = make(map[*Client]bool)
			}
			h.rooms[ev.chatID][ev.client] = true
			h.mu.Unlock()

		case ev := <-h.leaveRoom:
			h.mu.Lock()
			delete(ev.client.Chats, ev.chatID)
			if room, ok := h.rooms[ev.chatID]; ok {
				delete(room, ev.client)
				if len(room) == 0 {
					delete(h.rooms, ev.chatID)
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(chatID primitive.ObjectID, message models.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[chatID]; ok {
		for client := range room {
			select {
			case client.Send <- []byte(message.Content):
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastJSONToRoom broadcasts a JSON message to all clients in a chat room
func (h *Hub) BroadcastJSONToRoom(chatID primitive.ObjectID, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
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

// BroadcastJSONToRoomExcludingSender broadcasts a JSON message to all clients in a chat room except the sender
func (h *Hub) BroadcastJSONToRoomExcludingSender(chatID primitive.ObjectID, senderID primitive.ObjectID, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[chatID]; ok {
		for client := range room {
			if client.ID == senderID {
				continue // Skip sender
			}
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}

// SendJSONToUser sends a JSON message to all active connections of a user.
func (h *Hub) SendJSONToUser(userID primitive.ObjectID, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.ID != userID {
			continue
		}
		select {
		case client.Send <- data:
		default:
			// If the client's send buffer is full, drop the message rather than blocking.
		}
	}
}

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID primitive.ObjectID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.onlineUsers[userID]
}

// GetOnlineUsers returns list of online user IDs
func (h *Hub) GetOnlineUsers() []primitive.ObjectID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	online := make([]primitive.ObjectID, 0, len(h.onlineUsers))
	for userID := range h.onlineUsers {
		online = append(online, userID)
	}
	return online
}





