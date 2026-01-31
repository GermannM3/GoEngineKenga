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

	// TODO: в следующей итерации сюда можно добавить буфер
	// событий коллизий/selection, чтобы отправлять их по WebSocket.
}

// NewManager создаёт менеджер API для заданного рантайма.
func NewManager(rt *runtime.Runtime, projectDir string) *Manager {
	return &Manager{
		rt:         rt,
		projectDir: projectDir,
		queue:      newCommandQueue(512),
		connections: make(map[string]ClientConnection),
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


