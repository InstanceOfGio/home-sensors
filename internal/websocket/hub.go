package hub

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Update struct {
	DeviceID    int64     `json:"device_id"`
	Mac         string    `json:"mac"`
	Name        string    `json:"name"`
	Room        string    `json:"room"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Battery     int64     `json:"battery"`
	Rssi        int64     `json:"rssi"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Hub struct {
	log      *slog.Logger
	upgrader websocket.Upgrader

	mu      sync.Mutex
	clients map[*websocket.Conn]chan Update
}

func New(log *slog.Logger) *Hub {
	return &Hub{
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		clients: make(map[*websocket.Conn]chan Update),
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws upgrade failed", "err", err)
		return
	}

	ch := make(chan Update, 16)
	h.mu.Lock()
	h.clients[conn] = ch
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		close(ch)
		conn.Close()
	}()

	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				conn.Close()
				return
			}
		}
	}()

	for update := range ch {
		if err := conn.WriteJSON(update); err != nil {
			return
		}
	}
}

func (h *Hub) Broadcast(u Update) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn, ch := range h.clients {
		select {
		case ch <- u:
		default:
			h.log.Warn("ws client too slow, dropping update", "remote", conn.RemoteAddr())
		}
	}
}
