package ipc

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/pool"
	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	// Re-export for backward compatibility
	ErrInvalidRequestID = errors.ErrInvalidRequestID
)

func validateJSONPayload(payload []byte) error {
	if payload == nil {
		return nil
	}

	if len(payload) == 0 {
		return errors.ErrInvalidJSON
	}

	if !utf8.Valid(payload) {
		return errors.ErrInvalidJSON
	}

	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return errors.ErrInvalidJSON
	}

	return nil
}

type Handler struct {
	pool *pool.Pool
}

func NewHandler(p *pool.Pool) *Handler {
	return &Handler{
		pool: p,
	}
}

func (h *Handler) Handle(frame *RequestFrame) *ResponseFrame {
	response := &ResponseFrame{
		RequestID: frame.RequestID,
	}

	switch frame.Command {
	case CmdOpenDB:
		if len(frame.Ops) == 0 || len(frame.Ops[0].Payload) == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid database name")
			return response
		}

		dbName := string(frame.Ops[0].Payload)
		dbID, err := h.pool.OpenOrCreateDB(dbName)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		response.Status = types.StatusOK
		response.Data = make([]byte, 8)
		binary.LittleEndian.PutUint64(response.Data, dbID)

	case CmdCloseDB:
		if frame.DBID == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid database ID")
			return response
		}

		if err := h.pool.CloseDB(frame.DBID); err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		response.Status = types.StatusOK

	case CmdExecute:
		if frame.DBID == 0 || len(frame.Ops) == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid request")
			return response
		}

		responses := make([][]byte, len(frame.Ops))
		for i, op := range frame.Ops {
			if op.OpType == types.OpCreate || op.OpType == types.OpUpdate {
				if err := validateJSONPayload(op.Payload); err != nil {
					responses[i] = []byte(err.Error())
					if response.Status == types.StatusOK {
						response.Status = types.StatusError
					}
					continue
				}
			}

			req := &pool.Request{
				DBID:     frame.DBID,
				DocID:    op.DocID,
				OpType:   op.OpType,
				Payload:  op.Payload,
				Response: make(chan pool.Response, 1),
			}

			h.pool.Execute(req)
			resp := <-req.Response

			if resp.Error != nil {
				responses[i] = []byte(resp.Error.Error())
			} else if resp.Data != nil {
				responses[i] = resp.Data
			}

			if resp.Status != types.StatusOK && response.Status == types.StatusOK {
				response.Status = resp.Status
			}
		}

		response.Status = types.StatusOK
		response.Data = serializeResponses(responses)

	case CmdStats:
		stats := h.pool.Stats()
		response.Status = types.StatusOK
		response.Data = serializeStats(stats)

	default:
		response.Status = types.StatusError
	}

	return response
}

func serializeResponses(responses [][]byte) []byte {
	var size uint32 = 4

	for _, resp := range responses {
		size += 4 + uint32(len(resp))
	}

	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:], uint32(len(responses)))

	offset := 4
	for _, resp := range responses {
		binary.LittleEndian.PutUint32(buf[offset:], uint32(len(resp)))
		offset += 4
		copy(buf[offset:], resp)
		offset += len(resp)
	}

	return buf
}

func serializeStats(stats *types.Stats) []byte {
	buf := make([]byte, 40)

	binary.LittleEndian.PutUint64(buf[0:], uint64(stats.TotalDBs))
	binary.LittleEndian.PutUint64(buf[8:], uint64(stats.ActiveDBs))
	binary.LittleEndian.PutUint64(buf[16:], stats.TotalTxns)
	binary.LittleEndian.PutUint64(buf[24:], stats.WALSize)
	binary.LittleEndian.PutUint64(buf[32:], stats.MemoryUsed)

	return buf
}

func readFrame(conn io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBuf)
	if length > MaxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func writeFrame(conn io.Writer, data []byte) error {
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}

	return nil
}
