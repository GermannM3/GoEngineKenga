package api

import (
	"encoding/json"
	"sync"

	"goenginekenga/engine/runtime"
)

// ClientConnection представляет собой соединение с клиентом
type ClientConnection interface {
	WriteJSON(v interface{}) error
}

// Manager связывает очередь команд и рантайм движка.
// Он живёт в игровом треде: ProcessPending вызывается из Update().
type Manager struct {
	rt         *runtime.Runtime
	projectDir string
	queue      *commandQueue

	// connections хранит активные соединения для отправки ответов
	connectionsMu sync.RWMutex
	connections   map[string]ClientConnection // key could be connection ID

	// viewportSubscribers — подписчики на стрим viewport frames
	viewportSubscribersMu sync.RWMutex
	viewportSubscribers  map[string]bool

	// TODO: в следующей итерации сюда можно добавить буфер
	// событий коллизий/selection, чтобы отправлять их по WebSocket.
}

// NewManager создаёт менеджер API для заданного рантайма.
func NewManager(rt *runtime.Runtime, projectDir string) *Manager {
	return &Manager{
		rt:                  rt,
		projectDir:          projectDir,
		queue:               newCommandQueue(512),
		connections:         make(map[string]ClientConnection),
		viewportSubscribers: make(map[string]bool),
	}
}

// EnqueueCommand используется WebSocket-сервером из сетевых горутин.
func (m *Manager) EnqueueCommand(cmd CommandEnvelope) {
	if m == nil {
		return
	}
	m.queue.Enqueue(cmd)
}

// RegisterConnection регистрирует новое соединение
func (m *Manager) RegisterConnection(id string, conn ClientConnection) {
	if m == nil {
		return
	}
	m.connectionsMu.Lock()
	defer m.connectionsMu.Unlock()
	m.connections[id] = conn
}

// UnregisterConnection удаляет соединение
func (m *Manager) UnregisterConnection(id string) {
	if m == nil {
		return
	}
	m.connectionsMu.Lock()
	defer m.connectionsMu.Unlock()
	delete(m.connections, id)
	m.viewportSubscribersMu.Lock()
	delete(m.viewportSubscribers, id)
	m.viewportSubscribersMu.Unlock()
}

// SubscribeViewport помечает соединение как подписчика на viewport frames
func (m *Manager) SubscribeViewport(connID string) {
	if m == nil {
		return
	}
	m.viewportSubscribersMu.Lock()
	defer m.viewportSubscribersMu.Unlock()
	m.viewportSubscribers[connID] = true
}

// UnsubscribeViewport снимает подписку на viewport
func (m *Manager) UnsubscribeViewport(connID string) {
	if m == nil {
		return
	}
	m.viewportSubscribersMu.Lock()
	defer m.viewportSubscribersMu.Unlock()
	delete(m.viewportSubscribers, connID)
}

// HasViewportSubscribers возвращает true, если есть подписчики на viewport
func (m *Manager) HasViewportSubscribers() bool {
	if m == nil {
		return false
	}
	m.viewportSubscribersMu.RLock()
	n := len(m.viewportSubscribers)
	m.viewportSubscribersMu.RUnlock()
	return n > 0
}

// BroadcastViewportFrame отправляет base64 PNG frame всем viewport subscribers
func (m *Manager) BroadcastViewportFrame(pngBase64 string) {
	if m == nil {
		return
	}
	m.viewportSubscribersMu.RLock()
	subs := make([]string, 0, len(m.viewportSubscribers))
	for id := range m.viewportSubscribers {
		subs = append(subs, id)
	}
	m.viewportSubscribersMu.RUnlock()

	if len(subs) == 0 {
		return
	}

	payload, _ := json.Marshal(map[string]string{"frame": pngBase64})
	resp := ResponseEnvelope{OK: true, Event: "viewport_frame", Data: payload}

	m.connectionsMu.RLock()
	defer m.connectionsMu.RUnlock()
	for _, id := range subs {
		if conn, ok := m.connections[id]; ok {
			_ = conn.WriteJSON(resp)
		}
	}
}

// SendResponse отправляет ответ на запрос
func (m *Manager) SendResponse(requestID, cmd string, data json.RawMessage, ok bool, errStr string) {
	m.connectionsMu.RLock()
	defer m.connectionsMu.RUnlock()

	// В простой реализации отправляем всем подключенным клиентам
	// В реальной системе нужно отслеживать, какому клиенту что отправлять
	for _, conn := range m.connections {
		resp := ResponseEnvelope{
			OK:        ok,
			Cmd:       cmd,
			RequestID: requestID,
			Error:     errStr,
			Data:      data,
		}
		_ = conn.WriteJSON(resp)
	}
}


