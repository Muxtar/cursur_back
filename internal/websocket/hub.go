package websocket

import (
	"chat-backend/internal/models"

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

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	rooms      map[primitive.ObjectID]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[primitive.ObjectID]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			for chatID := range client.Chats {
				if h.rooms[chatID] == nil {
					h.rooms[chatID] = make(map[*Client]bool)
				}
				h.rooms[chatID][client] = true
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				for chatID := range client.Chats {
					if room, ok := h.rooms[chatID]; ok {
						delete(room, client)
						if len(room) == 0 {
							delete(h.rooms, chatID)
						}
					}
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





