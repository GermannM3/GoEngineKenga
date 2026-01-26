package asset

import (
	"context"
	"sync"
	"time"
)

// LoadRequest — запрос на загрузку ассета (AAA: async, без блокировки кадра).
type LoadRequest struct {
	ID   string
	Path string
	Type string // mesh, texture, audio, etc.
}

// LoadResult — результат загрузки.
type LoadResult struct {
	Request LoadRequest
	Mesh    *Mesh
	Error   error
}

// AsyncLoader загружает ассеты в фоне.
type AsyncLoader struct {
	mu      sync.Mutex
	project string
	queue   []LoadRequest
	results chan LoadResult
	run     bool
}

// NewAsyncLoader создаёт асинхронный загрузчик.
func NewAsyncLoader(projectDir string) *AsyncLoader {
	return &AsyncLoader{
		project: projectDir,
		results: make(chan LoadResult, 32),
	}
}

// Enqueue добавляет запрос в очередь.
func (a *AsyncLoader) Enqueue(req LoadRequest) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queue = append(a.queue, req)
}

// Start запускает фоновую загрузку.
func (a *AsyncLoader) Start(ctx context.Context) {
	a.mu.Lock()
	if a.run {
		a.mu.Unlock()
		return
	}
	a.run = true
	a.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.mu.Lock()
				a.run = false
				a.mu.Unlock()
				return
			default:
				a.mu.Lock()
				if len(a.queue) == 0 {
					a.mu.Unlock()
					time.Sleep(10 * time.Millisecond)
					continue
				}
				req := a.queue[0]
				a.queue = a.queue[1:]
				a.mu.Unlock()

				var res LoadResult
				res.Request = req
				if req.Type == "mesh" && req.Path != "" {
					m, err := LoadMesh(req.Path)
					res.Mesh = m
					res.Error = err
				}

				select {
				case a.results <- res:
				default:
					// channel full, drop or retry
				}
			}
		}
	}()
}

// Stop останавливает загрузчик.
func (a *AsyncLoader) Stop() {
	a.mu.Lock()
	a.run = false
	a.mu.Unlock()
}

// Poll возвращает следующий результат, если есть (неблокирующий).
func (a *AsyncLoader) Poll() (LoadResult, bool) {
	select {
	case r := <-a.results:
		return r, true
	default:
		return LoadResult{}, false
	}
}
