package api

import (
	"sync"
)

// commandQueue — простая потокобезопасная очередь команд.
// В неё пишут WebSocket-горутины, а читает игровой тред в Update().
type commandQueue struct {
	mu   sync.Mutex
	buf  []CommandEnvelope
	cap  int
}

func newCommandQueue(capacity int) *commandQueue {
	if capacity <= 0 {
		capacity = 256
	}
	return &commandQueue{
		buf: make([]CommandEnvelope, 0, capacity),
		cap: capacity,
	}
}

// Enqueue добавляет команду в очередь.
// Если очередь переполнена, старая голова отбрасывается (ring-buffer стиль),
// чтобы не блокировать сеть.
func (q *commandQueue) Enqueue(cmd CommandEnvelope) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.buf) < q.cap {
		q.buf = append(q.buf, cmd)
		return
	}

	// Переполнение: сдвигаем окно и добавляем в конец.
	copy(q.buf[0:], q.buf[1:])
	q.buf[len(q.buf)-1] = cmd
}

// Drain забирает все команды одним срезом и очищает очередь.
func (q *commandQueue) Drain() []CommandEnvelope {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.buf) == 0 {
		return nil
	}
	out := make([]CommandEnvelope, len(q.buf))
	copy(out, q.buf)
	q.buf = q.buf[:0]
	return out
}

