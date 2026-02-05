package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/kartikbazzad/bunbase/buncast/internal/broker"
	"github.com/kartikbazzad/bunbase/buncast/internal/config"
	"github.com/kartikbazzad/bunbase/buncast/internal/logger"
)

// Handler handles IPC requests using the broker.
type Handler struct {
	broker *broker.Broker
	cfg    *config.Config
	logger *logger.Logger
}

// NewHandler creates a new IPC handler.
func NewHandler(b *broker.Broker, cfg *config.Config, log *logger.Logger) *Handler {
	return &Handler{
		broker: b,
		cfg:    cfg,
		logger: log,
	}
}

// SubscribeSession is returned for CmdSubscribe so the server can unregister on disconnect.
type SubscribeSession struct {
	Topic     string
	Sub       broker.Subscriber
	CloseChan chan struct{} // Closed when connection is closed (write error detected)
}

// Handle processes a request and returns a response and optional subscribe session.
// For CmdSubscribe, the session is non-nil and the server must Unsubscribe when the connection closes.
func (h *Handler) Handle(conn net.Conn, req *RequestFrame) (resp *ResponseFrame, session *SubscribeSession, err error) {
	resp = &ResponseFrame{RequestID: req.RequestID}

	switch req.Command {
	case CmdCreateTopic:
		r, _ := h.handleCreateTopic(req, resp)
		return r, nil, nil
	case CmdDeleteTopic:
		r, _ := h.handleDeleteTopic(req, resp)
		return r, nil, nil
	case CmdListTopics:
		r, _ := h.handleListTopics(req, resp)
		return r, nil, nil
	case CmdPublish:
		r, _ := h.handlePublish(req, resp)
		return r, nil, nil
	case CmdSubscribe:
		r, sess := h.handleSubscribe(conn, req, resp)
		return r, sess, nil
	default:
		resp.Status = StatusError
		resp.Payload = []byte(`{"error":"unknown command"}`)
		return resp, nil, nil
	}
}

func (h *Handler) handleCreateTopic(req *RequestFrame, resp *ResponseFrame) (*ResponseFrame, error) {
	topic, err := DecodeTopicPayload(req.Payload)
	if err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return resp, nil
	}
	h.broker.CreateTopic(topic)
	resp.Status = StatusOK
	resp.Payload = []byte("{}")
	return resp, nil
}

func (h *Handler) handleDeleteTopic(req *RequestFrame, resp *ResponseFrame) (*ResponseFrame, error) {
	topic, err := DecodeTopicPayload(req.Payload)
	if err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return resp, nil
	}
	h.broker.DeleteTopic(topic)
	resp.Status = StatusOK
	resp.Payload = []byte("{}")
	return resp, nil
}

func (h *Handler) handleListTopics(req *RequestFrame, resp *ResponseFrame) (*ResponseFrame, error) {
	topics := h.broker.ListTopics()
	payload, err := EncodeListTopicsResponse(topics)
	if err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return resp, nil
	}
	resp.Status = StatusOK
	resp.Payload = payload
	return resp, nil
}

func (h *Handler) handlePublish(req *RequestFrame, resp *ResponseFrame) (*ResponseFrame, error) {
	topic, body, err := DecodePublishPayload(req.Payload)
	if err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return resp, nil
	}
	h.broker.CreateTopic(topic)
	h.logger.Info("IPC publish topic=%s subscribers=%d payload_bytes=%d", topic, h.broker.SubscriberCount(topic), len(body))
	msg := &broker.Message{Topic: topic, Payload: body}
	h.broker.Publish(msg)
	resp.Status = StatusOK
	resp.Payload = []byte("{}")
	return resp, nil
}

// connSubscriber implements broker.Subscriber by writing message frames to a connection.
type connSubscriber struct {
	conn      net.Conn
	mu        sync.Mutex
	logger    *logger.Logger
	closeChan chan struct{} // Closed when write fails (connection closed)
}

func (c *connSubscriber) Send(msg *broker.Message) {
	frame, err := EncodeMessageFrame(msg.Topic, msg.Payload)
	if err != nil {
		c.logger.Error("encode message frame: %v", err)
		return
	}
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(frame)))
	c.mu.Lock()
	_, err = c.conn.Write(lenBuf)
	if err == nil {
		_, err = c.conn.Write(frame)
	}
	c.mu.Unlock()
	if err != nil {
		// Connection closed or error - signal close
		select {
		case <-c.closeChan:
			// Already closed
		default:
			close(c.closeChan)
		}
		c.logger.Debug("subscriber write error (client likely disconnected): %v", err)
	}
}

func (h *Handler) handleSubscribe(conn net.Conn, req *RequestFrame, resp *ResponseFrame) (*ResponseFrame, *SubscribeSession) {
	topic, err := DecodeTopicPayload(req.Payload)
	if err != nil {
		resp.Status = StatusError
		resp.Payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return resp, nil
	}
	h.broker.CreateTopic(topic)
	closeChan := make(chan struct{})
	sub := &connSubscriber{conn: conn, logger: h.logger, closeChan: closeChan}
	h.broker.Subscribe(topic, sub)
	h.logger.Info("IPC subscribe topic=%s subscribers=%d", topic, h.broker.SubscriberCount(topic))
	resp.Status = StatusOK
	resp.Payload = []byte("{}")
	return resp, &SubscribeSession{Topic: topic, Sub: sub, CloseChan: closeChan}
}

// ErrorPayload returns a JSON error payload.
func ErrorPayload(msg string) []byte {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return b
}
