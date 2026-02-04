package logstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LokiStore implements Store by pushing logs to Loki and querying via Loki API.
type LokiStore struct {
	baseURL    string
	httpClient *http.Client
}

// NewLokiStore creates a Loki-backed log store. baseURL is the Loki HTTP API base (e.g. http://loki:3100).
func NewLokiStore(baseURL string) *LokiStore {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &LokiStore{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// lokiPushRequest is the payload for POST /loki/api/v1/push
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"` // [[timestamp_ns, line], ...]
}

// Append sends a log line to Loki.
func (s *LokiStore) Append(functionID, invocationID, level, message string) error {
	if functionID == "" {
		functionID = "unknown"
	}
	if level == "" {
		level = "info"
	}
	ts := time.Now().UnixNano()
	tsStr := strconv.FormatInt(ts, 10)
	// Escape labels: Loki allows only [a-zA-Z0-9_] in label values; replace invalid with underscore
	stream := map[string]string{
		"function_id":   sanitizeLabel(functionID),
		"level":         sanitizeLabel(level),
		"invocation_id": sanitizeLabel(invocationID),
	}
	body := lokiPushRequest{
		Streams: []lokiStream{{
			Stream: stream,
			Values: [][]string{{tsStr, message}},
		}},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("loki push marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/loki/api/v1/push", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("loki push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("loki push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki push status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sanitizeLabel(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	if len(s) > 1024 {
		return s[:1024]
	}
	return s
}

// lokiQueryResponse is the response from GET /loki/api/v1/query_range
type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string       `json:"values"` // [[ts_ns, line], ...]
		} `json:"result"`
	} `json:"data"`
}

// GetLogs queries Loki for logs of the given function since the given time, up to limit entries.
func (s *LokiStore) GetLogs(functionID string, since time.Time, limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	// LogQL: {function_id="..."} with optional time range
	query := fmt.Sprintf(`{function_id=%q}`, sanitizeLabel(functionID))
	start := since.UnixNano()
	end := time.Now().UnixNano()
	u, err := url.Parse(s.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start, 10))
	q.Set("end", strconv.FormatInt(end, 10))
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("loki query request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loki query: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki query status %d: %s", resp.StatusCode, string(b))
	}
	var out lokiQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("loki query decode: %w", err)
	}
	var entries []LogEntry
	for _, res := range out.Data.Result {
		stream := res.Stream
		invID := stream["invocation_id"]
		level := stream["level"]
		if level == "" {
			level = "info"
		}
		for _, pair := range res.Values {
			if len(pair) != 2 {
				continue
			}
			tsNs, line := pair[0], pair[1]
			ts, _ := strconv.ParseInt(tsNs, 10, 64)
			entries = append(entries, LogEntry{
				FunctionID:   functionID,
				InvocationID: invID,
				Level:        level,
				Message:      line,
				CreatedAt:    time.Unix(0, ts),
			})
		}
	}
	// Sort by CreatedAt ascending (oldest first); Loki may return out of order
	// For simplicity we don't sort if order is acceptable from Loki
	return entries, nil
}
