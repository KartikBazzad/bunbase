package ipc

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"time"
	"unicode/utf8"

	"github.com/kartikbazzad/docdb/internal/errors"
	"github.com/kartikbazzad/docdb/internal/metrics"
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
	pool     *pool.Pool
	exporter *metrics.PrometheusExporter
}

func NewHandler(p *pool.Pool) *Handler {
	return &Handler{
		pool:     p,
		exporter: metrics.NewPrometheusExporter(),
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

		startTime := time.Now()
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
				DBID:       frame.DBID,
				Collection: op.Collection,
				DocID:      op.DocID,
				OpType:     op.OpType,
				Payload:    op.Payload,
				PatchOps:   op.PatchOps,
				Response:   make(chan pool.Response, 1),
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

		duration := time.Since(startTime)
		statusStr := "ok"
		if response.Status != types.StatusOK {
			statusStr = "error"
		}
		h.exporter.RecordOperation("execute", statusStr, duration)

		response.Status = types.StatusOK
		response.Data = serializeResponses(responses)

	case CmdQuery:
		if frame.DBID == 0 || len(frame.Ops) == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid query: need DBID and at least one op with collection")
			return response
		}
		collection := frame.Ops[0].Collection
		if collection == "" {
			collection = "_default"
		}
		queryPayload := frame.Ops[0].Payload
		data, err := h.pool.ExecuteQuery(frame.DBID, collection, queryPayload)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}
		response.Status = types.StatusOK
		response.Data = data

	case CmdCreateCollection:
		if frame.DBID == 0 || len(frame.Ops) == 0 || frame.Ops[0].Collection == "" {
			response.Status = types.StatusError
			response.Data = []byte("invalid collection name")
			return response
		}

		err := h.pool.CreateCollection(frame.DBID, frame.Ops[0].Collection)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		response.Status = types.StatusOK

	case CmdDeleteCollection:
		if frame.DBID == 0 || len(frame.Ops) == 0 || frame.Ops[0].Collection == "" {
			response.Status = types.StatusError
			response.Data = []byte("invalid collection name")
			return response
		}

		err := h.pool.DeleteCollection(frame.DBID, frame.Ops[0].Collection)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		response.Status = types.StatusOK

	case CmdListCollections:
		if frame.DBID == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid database ID")
			return response
		}

		collections, err := h.pool.ListCollections(frame.DBID)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		collectionsJSON, err := json.Marshal(collections)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte("failed to serialize collections")
			return response
		}

		response.Status = types.StatusOK
		response.Data = collectionsJSON

	case CmdListDBs:
		dbInfos := h.pool.ListDBs()
		dbInfosJSON, err := json.Marshal(dbInfos)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte("failed to serialize database info")
			return response
		}

		response.Status = types.StatusOK
		response.Data = dbInfosJSON

	case CmdStats:
		stats := h.pool.Stats()
		response.Status = types.StatusOK
		response.Data = serializeStats(stats)

	case CmdMetrics:
		stats := h.pool.Stats()
		// Update exporter with current stats
		h.exporter.SetDocumentsTotal(stats.DocsLive)
		h.exporter.SetMemoryBytes(stats.MemoryUsed)
		h.exporter.SetWALSizeBytes(stats.WALSize)

		// Record errors from error tracker
		errorTracker := h.pool.GetErrorTracker()
		for category := errors.ErrorTransient; category <= errors.ErrorNetwork; category++ {
			count := errorTracker.GetErrorCount(category)
			if count > 0 {
				for i := uint64(0); i < count; i++ {
					h.exporter.RecordError(category)
				}
			}
		}

		metricsOutput := h.exporter.Export(stats)
		response.Status = types.StatusOK
		response.Data = []byte(metricsOutput)

	case CmdHeal:
		if frame.DBID == 0 || len(frame.Ops) == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid request: db_id and doc_id required")
			return response
		}

		op := frame.Ops[0]
		err := h.pool.HealDocument(frame.DBID, op.Collection, op.DocID)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		// Record healing metrics
		h.exporter.RecordHealingOperation(1)

		response.Status = types.StatusOK
		response.Data = []byte("OK")

	case CmdHealAll:
		if frame.DBID == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid request: db_id required")
			return response
		}

		healed, err := h.pool.HealAll(frame.DBID)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		// Record healing metrics
		h.exporter.RecordHealingOperation(uint64(len(healed)))

		// Serialize healed document IDs
		healedJSON, err := json.Marshal(healed)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte("failed to serialize healed documents")
			return response
		}

		response.Status = types.StatusOK
		response.Data = healedJSON

	case CmdHealStats:
		if frame.DBID == 0 {
			response.Status = types.StatusError
			response.Data = []byte("invalid request: db_id required")
			return response
		}

		stats, err := h.pool.HealStats(frame.DBID)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte(err.Error())
			return response
		}

		// Convert HealingStats to a map for JSON serialization
		statsMap := map[string]interface{}{
			"TotalScans":         stats.TotalScans,
			"DocumentsHealed":    stats.DocumentsHealed,
			"DocumentsCorrupted": stats.DocumentsCorrupted,
			"OnDemandHealings":   stats.OnDemandHealings,
			"BackgroundHealings": stats.BackgroundHealings,
		}

		// Format time fields as RFC3339 strings
		if !stats.LastScanTime.IsZero() {
			statsMap["LastScanTime"] = stats.LastScanTime.Format("2006-01-02T15:04:05Z07:00")
		} else {
			statsMap["LastScanTime"] = ""
		}

		if !stats.LastHealingTime.IsZero() {
			statsMap["LastHealingTime"] = stats.LastHealingTime.Format("2006-01-02T15:04:05Z07:00")
		} else {
			statsMap["LastHealingTime"] = ""
		}

		// Serialize healing stats
		statsJSON, err := json.Marshal(statsMap)
		if err != nil {
			response.Status = types.StatusError
			response.Data = []byte("failed to serialize healing stats")
			return response
		}

		response.Status = types.StatusOK
		response.Data = statsJSON

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
