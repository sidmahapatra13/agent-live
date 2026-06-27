package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub maintains a set of active WebSocket connections and broadcasts events.
// It also keeps a rolling history of recent events for replay to new clients.
type Hub struct {
	mu       sync.RWMutex
	clients  map[*websocket.Conn]bool
	history  [][]byte // rolling buffer of recent events
	upgrader websocket.Upgrader
}

const maxHistory = 500

// NewHub creates a new WebSocket hub.
// allowedOrigin restricts which Origin header is accepted on WebSocket upgrade.
// When empty, all origins are allowed (insecure — use only in dev).
func NewHub(allowedOrigin string) *Hub {
	h := &Hub{
		clients: make(map[*websocket.Conn]bool),
		history: make([][]byte, 0, maxHistory),
	}
	h.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if allowedOrigin == "" {
				return true // no restriction (dev mode)
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // allow non-browser clients (curl, code) without Origin
			}
			return origin == allowedOrigin
		},
	}
	return h
}

// HandleWS upgrades an HTTP connection to WebSocket.
// On connect, replays all history events so the client sees the full session.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	// Replay history to this new client
	for _, event := range h.history {
		if err := conn.WriteMessage(websocket.TextMessage, event); err != nil {
			log.Printf("WS replay error: %v", err)
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			return
		}
	}
	h.mu.Unlock()

	log.Printf("WS client connected (%d total, %d history events replayed)", len(h.clients), len(h.history))

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
		log.Printf("WS client disconnected (%d remaining)", len(h.clients))
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Broadcast sends a JSON message to all connected clients
// and appends it to the rolling history buffer.
func (h *Hub) Broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Store in history (rolling buffer)
	if len(h.history) >= maxHistory {
		h.history = append(h.history[1:], data)
	} else {
		h.history = append(h.history, data)
	}

	// Send to all connected clients
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WS write error: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
