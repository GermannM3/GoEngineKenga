package api

import "encoding/json"

// CommandEnvelope описывает входящую JSON-команду по WebSocket.
// Формат на проводе:
// {
//   "cmd": "load_model",
//   "request_id": "optional",
//   "data": { ... }
// }
type CommandEnvelope struct {
	Cmd       string          `json:"cmd"`
	RequestID string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	// ConnID заполняется сервером при чтении (для subscribe_viewport и т.п.)
	ConnID string `json:"-"`
}

// ResponseEnvelope описывает базовый ответ/событие.
// Для команд ok/err, для событий используем поле Event.
type ResponseEnvelope struct {
	OK        bool   `json:"ok"`
	Cmd       string `json:"cmd,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Error     string `json:"error,omitempty"`

	// Event используется для асинхронных нотификаций (collision, log и т.п.).
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

