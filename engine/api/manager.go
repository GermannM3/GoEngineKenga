package api

import (
	"goenginekenga/engine/runtime"
)

// Manager связывает очередь команд и рантайм движка.
// Он живёт в игровом треде: ProcessPending вызывается из Update().
type Manager struct {
	rt         *runtime.Runtime
	projectDir string
	queue      *commandQueue

	// TODO: в следующей итерации сюда можно добавить буфер
	// событий коллизий/selection, чтобы отправлять их по WebSocket.
}

// NewManager создаёт менеджер API для заданного рантайма.
func NewManager(rt *runtime.Runtime, projectDir string) *Manager {
	return &Manager{
		rt:         rt,
		projectDir: projectDir,
		queue:      newCommandQueue(512),
	}
}

// EnqueueCommand используется WebSocket-сервером из сетевых горутин.
func (m *Manager) EnqueueCommand(cmd CommandEnvelope) {
	if m == nil {
		return
	}
	m.queue.Enqueue(cmd)
}


