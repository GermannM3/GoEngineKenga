package network

import (
	"context"
	"net"
	"sync"
	"time"
)

// Transport — тип транспорта (AAA: TCP/UDP/WebRTC и т.д.).
type Transport string

const (
	TransportTCP Transport = "tcp"
	TransportUDP Transport = "udp"
)

// Session — сетевая сессия (клиент или серверная сторона соединения).
type Session struct {
	mu        sync.RWMutex
	conn      net.Conn
	id        string
	createdAt time.Time
	closed    bool
}

// SessionConfig — конфиг сессии.
type SessionConfig struct {
	Transport   Transport
	Address     string
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

// SessionHandler — обработчик событий сессии (подключение, данные, отключение).
type SessionHandler interface {
	OnConnect(s *Session)
	OnDisconnect(s *Session, err error)
	OnMessage(s *Session, data []byte)
}

// Client подключается к серверу и держит сессию.
type Client struct {
	mu      sync.RWMutex
	config  SessionConfig
	handler SessionHandler
	session *Session
}

// NewClient создаёт клиента.
func NewClient(config SessionConfig, handler SessionHandler) *Client {
	return &Client{config: config, handler: handler}
}

// Connect подключается к серверу.
func (c *Client) Connect(ctx context.Context) error {
	dialer := net.Dialer{Timeout: c.config.DialTimeout}
	conn, err := dialer.DialContext(ctx, string(c.config.Transport), c.config.Address)
	if err != nil {
		return err
	}

	s := &Session{conn: conn, id: "client", createdAt: time.Now()}
	c.mu.Lock()
	c.session = s
	c.mu.Unlock()

	if c.handler != nil {
		c.handler.OnConnect(s)
	}
	return nil
}

// Disconnect отключается.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	s := c.session
	c.session = nil
	c.mu.Unlock()

	if s == nil {
		return nil
	}
	err := s.Close()
	if c.handler != nil {
		c.handler.OnDisconnect(s, err)
	}
	return err
}

// Send отправляет данные.
func (c *Client) Send(data []byte) error {
	c.mu.RLock()
	s := c.session
	c.mu.RUnlock()

	if s == nil {
		return ErrNotConnected
	}
	return s.Send(data)
}

// ErrNotConnected — не подключено.
var ErrNotConnected = &ConnectionError{Msg: "not connected"}

// ConnectionError — ошибка соединения.
type ConnectionError struct {
	Msg string
}

func (e *ConnectionError) Error() string {
	return e.Msg
}

// ID возвращает идентификатор сессии.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Send отправляет сырые данные.
func (s *Session) Send(data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.conn == nil {
		return ErrNotConnected
	}
	_, err := s.conn.Write(data)
	return err
}

// Close закрывает сессию.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// Server принимает входящие соединения (заготовка под AAA-мультиплеер).
type Server struct {
	mu       sync.RWMutex
	config   SessionConfig
	handler  SessionHandler
	listener net.Listener
	sessions map[string]*Session
	done     chan struct{}
}

// NewServer создаёт сервер.
func NewServer(config SessionConfig, handler SessionHandler) *Server {
	return &Server{
		config:   config,
		handler:  handler,
		sessions: make(map[string]*Session),
		done:     make(chan struct{}),
	}
}

// Start запускает прослушивание.
func (sv *Server) Start() error {
	l, err := net.Listen(string(sv.config.Transport), sv.config.Address)
	if err != nil {
		return err
	}
	sv.mu.Lock()
	sv.listener = l
	sv.mu.Unlock()
	return nil
}

// Stop останавливает сервер.
func (sv *Server) Stop() error {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	close(sv.done)
	if sv.listener != nil {
		err := sv.listener.Close()
		sv.listener = nil
		return err
	}
	return nil
}
