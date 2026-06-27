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
	mu         sync.RWMutex
	clients    map[*websocket.Conn]bool
	history    [][]byte // rolling buffer of recent events
	historyMax int
	upgrader   websocket.Upgrader
}

// NewHub creates a new WebSocket hub.
// allowedOrigin restricts which Origin header is accepted on WebSocket upgrade.
// When empty, all origins are allowed (insecure — use only in dev).
func NewHub(allowedOrigin string, historyMax int) *Hub {
	if historyMax < 1 {
		historyMax = 500
	}
	h := &Hub{
		clients:    make(map[*websocket.Conn]bool),
		history:    make([][]byte, 0, historyMax),
		historyMax: historyMax,
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
			// Normalise: strip port/scheme for comparison
			if origin == allowedOrigin {
				return true
			}
			// Accept localhost variants as equivalent
			host := stripScheme(origin)
			return host == "localhost" || host == "127.0.0.1" || host == "[::1]"
		},
	}
	return h
}

// stripScheme removes http[s]:// prefix from a URL.
func stripScheme(s string) string {
	for _, p := range []string{"https://", "http://", "wss://", "ws://"} {
		if len(s) > len(p) && s[:len(p)] == p {
			s = s[len(p):]
		}
	}
	// Strip port suffix (IPv6-safe: only strip if :digit port)
	if idx := stringsLastIndexByte(s, ':'); idx > 0 {
		after := s[idx+1:]
		isPort := true
		for _, c := range after {
			if c < '0' || c > '9' {
				isPort = false
				break
			}
		}
		if isPort {
			s = s[:idx]
		}
	}
	return s
}

// stringsLastIndexByte avoids importing strings for one call.
func stringsLastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
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
	count := len(h.clients)
	histLen := len(h.history)
	h.mu.Unlock()

	log.Printf("WS client connected (%d total, %d history events replayed)", count, histLen)

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
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
// Client list is copied under lock; writes happen outside lock so
// one slow/dead client doesn't block all other broadcasts.
func (h *Hub) Broadcast(data []byte) {
	h.mu.Lock()
	// Store in history (rolling buffer)
	if len(h.history) >= h.historyMax {
		h.history = append(h.history[1:], data)
	} else {
		h.history = append(h.history, data)
	}
	// Copy client list so we can write outside lock
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.Unlock()

	// Write to all clients (outside lock)
	for _, conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WS write error: %v", err)
			conn.Close()
			h.removeClient(conn)
		}
	}
}

func (h *Hub) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}
