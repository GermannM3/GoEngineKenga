package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionInfo хранит информацию о соединении
type ConnectionInfo struct {
	conn *websocket.Conn
	id   string
}

// Server поднимает WebSocket-эндпойнт и прокидывает входящие команды
// в Manager. На этом этапе поддерживаем один простой эндпойнт /ws
// и ориентируемся на один основной клиент (Python/PyQt).
type Server struct {
	addr    string
	manager *Manager

	upgrader websocket.Upgrader

	httpSrv *http.Server
	once    sync.Once

	// connections хранит активные соединения
	connections   map[string]*ConnectionInfo
	connectionsMu sync.RWMutex
}

// NewServer создаёт WebSocket-сервер, но не запускает его.
func NewServer(addr string, manager *Manager) *Server {
	if addr == "" {
		addr = "127.0.0.1:7777"
	}
	return &Server{
		addr:    addr,
		manager: manager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				// Локальное управление, CORS/Origin не ограничиваем.
				return true
			},
		},
		connections: make(map[string]*ConnectionInfo),
	}
}

// Start поднимает HTTP-сервер и начинает принимать подключения.
// Контекст используется для мягкой остановки.
func (s *Server) Start(ctx context.Context) error {
	s.once.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", s.handleWS)

		s.httpSrv = &http.Server{
			Addr:    s.addr,
			Handler: mux,
		}

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.httpSrv.Shutdown(shutdownCtx)
		}()

		go func() {
			if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("api: WebSocket server error: %v", err)
			}
		}()
	})

	return nil
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("api: websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Генерируем ID для соединения (можно использовать более сложную логику)
	connID := fmt.Sprintf("%p", conn)

	// Регистрируем соединение
	s.connectionsMu.Lock()
	s.connections[connID] = &ConnectionInfo{
		conn: conn,
		id:   connID,
	}
	s.connectionsMu.Unlock()

	// Сообщаем менеджеру о новом соединении
	if s.manager != nil {
		s.manager.RegisterConnection(connID, conn)
	}

	defer func() {
		// Удаляем соединение при выходе
		s.connectionsMu.Lock()
		delete(s.connections, connID)
		if s.manager != nil {
			s.manager.UnregisterConnection(connID)
		}
		s.connectionsMu.Unlock()
	}()

	for {
		var envelope CommandEnvelope
		if err := conn.ReadJSON(&envelope); err != nil {
			// Клиент отключился или случилась ошибка чтения.
			// Логируем и выходим из цикла.
			// log.Printf("api: read error: %v", err)
			return
		}

		if s.manager != nil {
			s.manager.EnqueueCommand(envelope)
		}

		// Простое подтверждение приёма команды.
		resp := ResponseEnvelope{
			OK:        true,
			Cmd:       envelope.Cmd,
			RequestID: envelope.RequestID,
		}
		_ = conn.WriteJSON(resp)
	}
}

