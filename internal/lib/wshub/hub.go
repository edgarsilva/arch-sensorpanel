package wshub

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
)

type Hub struct {
	mu            sync.RWMutex
	settingsConns map[*websocket.Conn]*settingsClient
}

type settingsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func New() *Hub {
	return &Hub{settingsConns: make(map[*websocket.Conn]*settingsClient)}
}

func (h *Hub) AddSettingsWSConn(conn *websocket.Conn) {
	if h == nil || conn == nil {
		return
	}

	h.mu.Lock()
	h.settingsConns[conn] = &settingsClient{conn: conn}
	h.mu.Unlock()
}

func (h *Hub) DelSettingsWSConn(conn *websocket.Conn) {
	if h == nil || conn == nil {
		return
	}

	h.mu.Lock()
	delete(h.settingsConns, conn)
	h.mu.Unlock()
}

func (h *Hub) BroadcastSettingsUpdated(version int64) {
	if h == nil {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"type":    "settings.updated",
		"version": version,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := make([]*settingsClient, 0, len(h.settingsConns))
	for _, client := range h.settingsConns {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if client == nil || client.conn == nil {
			continue
		}

		client.writeMu.Lock()
		err := client.conn.WriteMessage(websocket.TextMessage, payload)
		client.writeMu.Unlock()
		if err != nil {
			h.DelSettingsWSConn(client.conn)
			_ = client.conn.Close()
		}
	}
}

func (h *Hub) Close() {
	if h == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn, client := range h.settingsConns {
		if client != nil && client.conn != nil {
			client.writeMu.Lock()
			_ = client.conn.Close()
			client.writeMu.Unlock()
		} else if conn != nil {
			_ = conn.Close()
		}
		delete(h.settingsConns, conn)
	}
}
