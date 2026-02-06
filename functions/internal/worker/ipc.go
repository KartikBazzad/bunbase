package worker

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Message types
const (
	MessageTypeReady    = "ready"
	MessageTypeInvoke   = "invoke"
	MessageTypeResponse = "response"
	MessageTypeLog      = "log"
	MessageTypeError    = "error"
)

// Message represents a JSON message in the IPC protocol
type Message struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ReadyPayload is sent by Bun worker when ready
type ReadyPayload struct{}

// InvokePayload is sent by Go to request function execution
type InvokePayload struct {
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	Headers       map[string]string `json:"headers"`
	Query         map[string]string `json:"query"`
	Body          string            `json:"body"`         // base64-encoded
	DeadlineMS    int64             `json:"deadline_ms"`  // per-invocation deadline
	ProjectID     string            `json:"project_id"`   // optional: project ID for admin context
	ProjectAPIKey string            `json:"project_api_key"` // optional: project public API key
	GatewayURL    string            `json:"gateway_url"`  // optional: gateway base URL
}

// ResponsePayload is sent by Bun worker after successful execution
type ResponsePayload struct {
	Status  int               `json:"status"`
	Headers map[string]string  `json:"headers"`
	Body    string            `json:"body"` // base64-encoded
}

// LogPayload is sent by Bun worker for log messages
type LogPayload struct {
	Level    string                 `json:"level"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorPayload is sent by Bun worker when handler execution fails
type ErrorPayload struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	Code    string `json:"code,omitempty"`
}

// MessageReader reads NDJSON messages from an io.Reader
type MessageReader struct {
	scanner *bufio.Scanner
}

// NewMessageReader creates a new message reader
func NewMessageReader(r io.Reader) *MessageReader {
	return &MessageReader{
		scanner: bufio.NewScanner(r),
	}
}

// Read reads the next message from the stream
// This blocks until a complete line is available
func (mr *MessageReader) Read() (*Message, error) {
	if !mr.scanner.Scan() {
		if err := mr.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	line := mr.scanner.Bytes()
	if len(line) == 0 {
		// Empty line, try again
		return mr.Read()
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// MessageWriter writes NDJSON messages to an io.Writer
type MessageWriter struct {
	writer io.Writer
	mu     chan struct{} // Simple mutex using channel
}

// NewMessageWriter creates a new message writer
func NewMessageWriter(w io.Writer) *MessageWriter {
	return &MessageWriter{
		writer: w,
		mu:     make(chan struct{}, 1),
	}
}

// Write writes a message to the stream
func (mw *MessageWriter) Write(msg *Message) error {
	mw.mu <- struct{}{} // Lock
	defer func() { <-mw.mu }() // Unlock

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	data = append(data, '\n')
	if _, err := mw.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// WriteReady writes a READY message
func (mw *MessageWriter) WriteReady(id string) error {
	payload, _ := json.Marshal(ReadyPayload{})
	return mw.Write(&Message{
		ID:      id,
		Type:    MessageTypeReady,
		Payload: payload,
	})
}

// WriteInvoke writes an INVOKE message
func (mw *MessageWriter) WriteInvoke(id string, payload *InvokePayload) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return mw.Write(&Message{
		ID:      id,
		Type:    MessageTypeInvoke,
		Payload: payloadData,
	})
}

// WriteResponse writes a RESPONSE message
func (mw *MessageWriter) WriteResponse(id string, payload *ResponsePayload) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return mw.Write(&Message{
		ID:      id,
		Type:    MessageTypeResponse,
		Payload: payloadData,
	})
}

// WriteLog writes a LOG message
func (mw *MessageWriter) WriteLog(id string, payload *LogPayload) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return mw.Write(&Message{
		ID:      id,
		Type:    MessageTypeLog,
		Payload: payloadData,
	})
}

// WriteError writes an ERROR message
func (mw *MessageWriter) WriteError(id string, payload *ErrorPayload) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return mw.Write(&Message{
		ID:      id,
		Type:    MessageTypeError,
		Payload: payloadData,
	})
}

// EncodeBody encodes a byte slice to base64 string
func EncodeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(body)
}

// DecodeBody decodes a base64 string to byte slice
func DecodeBody(body string) ([]byte, error) {
	if body == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(body)
}

// ParseInvokePayload parses an InvokePayload from a message
func ParseInvokePayload(msg *Message) (*InvokePayload, error) {
	if msg.Type != MessageTypeInvoke {
		return nil, fmt.Errorf("expected invoke message, got %s", msg.Type)
	}
	var payload InvokePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ParseResponsePayload parses a ResponsePayload from a message
func ParseResponsePayload(msg *Message) (*ResponsePayload, error) {
	if msg.Type != MessageTypeResponse {
		return nil, fmt.Errorf("expected response message, got %s", msg.Type)
	}
	var payload ResponsePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ParseLogPayload parses a LogPayload from a message
func ParseLogPayload(msg *Message) (*LogPayload, error) {
	if msg.Type != MessageTypeLog {
		return nil, fmt.Errorf("expected log message, got %s", msg.Type)
	}
	var payload LogPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ParseErrorPayload parses an ErrorPayload from a message
func ParseErrorPayload(msg *Message) (*ErrorPayload, error) {
	if msg.Type != MessageTypeError {
		return nil, fmt.Errorf("expected error message, got %s", msg.Type)
	}
	var payload ErrorPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// CheckDeadline checks if the deadline has passed
func CheckDeadline(deadlineMS int64) bool {
	if deadlineMS <= 0 {
		return false
	}
	deadline := time.Unix(0, deadlineMS*int64(time.Millisecond))
	return time.Now().After(deadline)
}
