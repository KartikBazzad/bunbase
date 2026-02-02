package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bunder/internal/data_structures"
	"github.com/kartikbazzad/bunbase/bunder/internal/pubsub"
	"github.com/kartikbazzad/bunbase/bunder/internal/ttl"
)

// Handler executes RESP commands: GET/SET/DEL/EXISTS/KEYS/TTL/EXPIRE, List/Set/Hash ops, PING/QUIT.
// It holds the KV store, per-key Lists/Sets/Hashes (in-memory), optional TTL manager, and optional pubsub.
type Handler struct {
	kv       *data_structures.KVStore
	lists    map[string]*data_structures.List
	sets     map[string]*data_structures.Set
	hashes   map[string]*data_structures.Hash
	ttl      *ttl.Manager
	pubsub   *pubsub.PubSubManager
	listsMu  sync.RWMutex
	setsMu   sync.RWMutex
	hashesMu sync.RWMutex
}

// NewHandler creates a handler with the given KV store and optional TTL/pubsub.
func NewHandler(kv *data_structures.KVStore, ttlMgr *ttl.Manager, pubsubMgr *pubsub.PubSubManager) *Handler {
	return &Handler{
		kv:     kv,
		lists:  make(map[string]*data_structures.List),
		sets:   make(map[string]*data_structures.Set),
		hashes: make(map[string]*data_structures.Hash),
		ttl:    ttlMgr,
		pubsub: pubsubMgr,
	}
}

// HandleConnection handles one client connection (RESP loop).
func (h *Handler) HandleConnection(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		val, err := ReadRESP(br)
		if err != nil {
			if err != io.EOF {
				_ = WriteRESP(bw, fmt.Errorf("read: %v", err))
				_ = bw.Flush()
			}
			return
		}
		arr, ok := val.([]interface{})
		if !ok {
			_ = WriteRESP(bw, fmt.Errorf("ERR expected array"))
			_ = bw.Flush()
			continue
		}
		cmd, args, err := ParseCommand(arr)
		if err != nil {
			_ = WriteRESP(bw, fmt.Errorf("ERR %v", err))
			_ = bw.Flush()
			continue
		}
		result, err := h.Exec(cmd, args)
		if err != nil {
			_ = WriteRESP(bw, err)
		} else {
			_ = WriteRESP(bw, result)
		}
		_ = bw.Flush()
	}
}

// Exec executes one RESP command (cmd and args) and returns the value to send to the client, or an error.
// Dispatches to GET, SET, DEL, EXISTS, KEYS, TTL, EXPIRE, List/Set/Hash ops, PING, QUIT.
func (h *Handler) Exec(cmd string, args [][]byte) (interface{}, error) {
	switch cmd {
	case "GET":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		v := h.kv.Get(args[0])
		return v, nil
	case "SET":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		if err := h.kv.Set(args[0], args[1]); err != nil {
			return nil, err
		}
		if h.pubsub != nil {
			h.pubsub.PublishOperation(pubsub.Operation{Type: "SET", Key: string(args[0]), Value: args[1]})
		}
		return "OK", nil
	case "DEL":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ok, err := h.kv.Delete(args[0])
		if err != nil {
			return nil, err
		}
		if ok && h.pubsub != nil {
			h.pubsub.PublishOperation(pubsub.Operation{Type: "DEL", Key: string(args[0])})
		}
		if ok {
			return int64(1), nil
		}
		return int64(0), nil
	case "EXISTS":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		if h.kv.Exists(args[0]) {
			return int64(1), nil
		}
		return int64(0), nil
	case "KEYS":
		pattern := []byte("*")
		if len(args) == 1 {
			pattern = args[0]
		}
		keys := h.kv.Keys(pattern)
		out := make([]interface{}, len(keys))
		for i, k := range keys {
			out[i] = k
		}
		return out, nil
	case "TTL":
		if len(args) != 1 || h.ttl == nil {
			if len(args) != 1 {
				return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
			}
			return int64(-1), nil
		}
		sec := h.ttl.TTLSeconds(string(args[0]), time.Now())
		return int64(sec), nil
	case "EXPIRE":
		if len(args) != 2 || h.ttl == nil {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		sec, err := strconv.Atoi(string(args[1]))
		if err != nil || sec < 0 {
			return nil, fmt.Errorf("ERR invalid expire time")
		}
		h.ttl.Set(string(args[0]), time.Now().Add(time.Duration(sec)*time.Second))
		return int64(1), nil
	case "LPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		l := h.getList(string(args[0]))
		n := l.LPush(args[1:]...)
		return int64(n), nil
	case "RPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		l := h.getList(string(args[0]))
		n := l.RPush(args[1:]...)
		return int64(n), nil
	case "LPOP":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		l := h.getList(string(args[0]))
		v := l.LPop()
		return v, nil
	case "RPOP":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		l := h.getList(string(args[0]))
		v := l.RPop()
		return v, nil
	case "LRANGE":
		if len(args) != 3 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		start, _ := strconv.Atoi(string(args[1]))
		stop, _ := strconv.Atoi(string(args[2]))
		l := h.getList(string(args[0]))
		elems := l.LRange(start, stop)
		out := make([]interface{}, len(elems))
		for i, e := range elems {
			out[i] = e
		}
		return out, nil
	case "LLEN":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		l := h.getList(string(args[0]))
		return int64(l.LLen()), nil
	case "SADD":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		s := h.getSet(string(args[0]))
		n := s.SAdd(args[1:]...)
		return int64(n), nil
	case "SREM":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		s := h.getSet(string(args[0]))
		n := s.SRem(args[1:]...)
		return int64(n), nil
	case "SMEMBERS":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		s := h.getSet(string(args[0]))
		mem := s.SMembers()
		out := make([]interface{}, len(mem))
		for i, m := range mem {
			out[i] = m
		}
		return out, nil
	case "SISMEMBER":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		s := h.getSet(string(args[0]))
		if s.SIsMember(args[1]) {
			return int64(1), nil
		}
		return int64(0), nil
	case "SCARD":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		s := h.getSet(string(args[0]))
		return int64(s.SCard()), nil
	case "HSET":
		if len(args) < 3 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		n := ha.HSet(args[1], args[2])
		return int64(n), nil
	case "HGET":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		v := ha.HGet(args[1])
		return v, nil
	case "HGETALL":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		out := ha.HGetAll()
		arr := make([]interface{}, len(out))
		for i, o := range out {
			arr[i] = o
		}
		return arr, nil
	case "HDEL":
		if len(args) < 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		n := ha.HDel(args[1:]...)
		return int64(n), nil
	case "HEXISTS":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		if ha.HExists(args[1]) {
			return int64(1), nil
		}
		return int64(0), nil
	case "HLEN":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for '%s'", cmd)
		}
		ha := h.getHash(string(args[0]))
		return int64(ha.HLen()), nil
	case "PING":
		if len(args) == 0 {
			return "PONG", nil
		}
		return args[0], nil
	case "QUIT":
		return "OK", nil
	default:
		return nil, fmt.Errorf("ERR unknown command '%s'", strings.ToLower(cmd))
	}
}

func (h *Handler) getList(key string) *data_structures.List {
	h.listsMu.Lock()
	defer h.listsMu.Unlock()
	if l, ok := h.lists[key]; ok {
		return l
	}
	l := data_structures.NewList()
	h.lists[key] = l
	return l
}

func (h *Handler) getSet(key string) *data_structures.Set {
	h.setsMu.Lock()
	defer h.setsMu.Unlock()
	if s, ok := h.sets[key]; ok {
		return s
	}
	s := data_structures.NewSet()
	h.sets[key] = s
	return s
}

func (h *Handler) getHash(key string) *data_structures.Hash {
	h.hashesMu.Lock()
	defer h.hashesMu.Unlock()
	if ha, ok := h.hashes[key]; ok {
		return ha
	}
	ha := data_structures.NewHash()
	h.hashes[key] = ha
	return ha
}
