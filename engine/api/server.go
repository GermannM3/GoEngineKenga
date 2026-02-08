package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	version string

	upgrader websocket.Upgrader

	httpSrv *http.Server
	once    sync.Once

	// connections хранит активные соединения
	connections   map[string]*ConnectionInfo
	connectionsMu sync.RWMutex
}

// NewServer создаёт WebSocket-сервер, но не запускает его.
func NewServer(addr string, manager *Manager) *Server {
	return NewServerWithVersion(addr, manager, "")
}

// NewServerWithVersion создаёт сервер с указанной версией (для /version).
func NewServerWithVersion(addr string, manager *Manager, version string) *Server {
	if addr == "" {
		addr = "127.0.0.1:7777"
	}
	return &Server{
		addr:    addr,
		manager: manager,
		version: version,
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
// Если порт занят, пробует следующие (7777, 7778, ...).
// Контекст используется для мягкой остановки.
func (s *Server) Start(ctx context.Context) error {
	s.once.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", s.handleWS)
		mux.HandleFunc("/version", s.handleVersion)

		s.httpSrv = &http.Server{Handler: mux}

		listener := listenWithFallback(s.addr)
		if listener == nil {
			log.Printf("api: WebSocket server not started (all ports busy)")
			return
		}
		s.httpSrv.Addr = listener.Addr().String()
		log.Printf("api: WebSocket on %s", s.httpSrv.Addr)

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.httpSrv.Shutdown(shutdownCtx)
		}()

		go func() {
			if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Printf("api: WebSocket server error: %v", err)
			}
		}()
	})

	return nil
}

// listenWithFallback пробует слушать addr; при "address already in use" — следующий порт.
func listenWithFallback(addr string) net.Listener {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host, portStr = "127.0.0.1", "7777"
	}
	port, _ := strconv.Atoi(portStr)
	if port <= 0 {
		port = 7777
	}

	for attempt := 0; attempt < 10; attempt++ {
		tryAddr := net.JoinHostPort(host, strconv.Itoa(port+attempt))
		ln, err := net.Listen("tcp", tryAddr)
		if err != nil {
			if strings.Contains(err.Error(), "bind") || strings.Contains(err.Error(), "address already in use") {
				continue
			}
			log.Printf("api: listen %s: %v", tryAddr, err)
			continue
		}
		return ln
	}
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
			// log.Printf("api: read error: %v", err)
			return
		}

		envelope.ConnID = connID
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

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	v := s.version
	if v == "" {
		v = "dev"
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"version": v})
}

