package ws

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	clients    map[int]map[*Client]bool
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

var GlobalHub *Hub

func NewHub() *Hub {
	return &Hub{
		clients:    map[int]map[*Client]bool{},
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.projectID] == nil {
				h.clients[client.projectID] = map[*Client]bool{}
			}
			h.clients[client.projectID][client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if set, ok := h.clients[client.projectID]; ok {
				if _, exists := set[client]; exists {
					delete(set, client)
					close(client.send)
					if len(set) == 0 {
						delete(h.clients, client.projectID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (h *Hub) Publish(projectID int, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if set, ok := h.clients[projectID]; ok {
		for client := range set {
			select {
			case client.send <- data:
			default:

			}
		}
	}
}
