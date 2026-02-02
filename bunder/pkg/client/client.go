// Package client provides a Go client for the Bunder KV database over the RESP (Redis) protocol.
// Connect to a Bunder server, then use Get, Set, Delete, Exists, Keys, Ping, or Do for raw commands.
package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// Client is a Bunder client: sends RESP arrays (e.g. GET key), reads RESP responses.
// Safe for concurrent use; each Do/Get/Set holds a lock for the request/response pair.
type Client struct {
	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer
	mu   sync.Mutex
}

// Options configures the client.
type Options struct {
	Addr    string
	Timeout time.Duration
}

// DefaultOptions returns default client options.
func DefaultOptions(addr string) Options {
	return Options{
		Addr:    addr,
		Timeout: 5 * time.Second,
	}
}

// Connect establishes a connection to a Bunder server.
func Connect(ctx context.Context, opts Options) (*Client, error) {
	dialer := net.Dialer{Timeout: opts.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", opts.Addr)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
		br:   bufio.NewReader(conn),
		bw:   bufio.NewWriter(conn),
	}, nil
}

// do sends a RESP array of bulk strings (e.g. GET, key) and reads one RESP value in reply.
func (c *Client) do(args ...[]byte) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Write: *N\r\n $len\r\n arg\r\n ...
	if _, err := fmt.Fprintf(c.bw, "*%d\r\n", len(args)); err != nil {
		return nil, err
	}
	for _, a := range args {
		if _, err := fmt.Fprintf(c.bw, "$%d\r\n", len(a)); err != nil {
			return nil, err
		}
		if _, err := c.bw.Write(a); err != nil {
			return nil, err
		}
		if _, err := c.bw.Write([]byte("\r\n")); err != nil {
			return nil, err
		}
	}
	if err := c.bw.Flush(); err != nil {
		return nil, err
	}
	return c.readResp()
}

func (c *Client) readResp() (interface{}, error) {
	b, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case '+':
		line, err := readLine(c.br)
		return line, err
	case '-':
		line, err := readLine(c.br)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("resp error: %s", line)
	case ':':
		line, err := readLine(c.br)
		if err != nil {
			return nil, err
		}
		n, _ := strconv.ParseInt(string(line), 10, 64)
		return n, nil
	case '$':
		line, err := readLine(c.br)
		if err != nil {
			return nil, err
		}
		n, _ := strconv.Atoi(string(line))
		if n == -1 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(c.br, buf); err != nil {
			return nil, err
		}
		return buf[:n], nil
	case '*':
		line, err := readLine(c.br)
		if err != nil {
			return nil, err
		}
		count, _ := strconv.Atoi(string(line))
		out := make([]interface{}, 0, count)
		for i := 0; i < count; i++ {
			v, err := c.readResp()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("resp unknown type %c", b)
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		return line[:len(line)-2], nil
	}
	return line[:len(line)-1], nil
}

// Get returns the value for key, or nil if not set.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	v, err := c.do([]byte("GET"), []byte(key))
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", v)
	}
	return b, nil
}

// Set sets key to value.
func (c *Client) Set(ctx context.Context, key string, value []byte) error {
	_, err := c.do([]byte("SET"), []byte(key), value)
	return err
}

// Delete removes key; returns true if the key was present.
func (c *Client) Delete(ctx context.Context, key string) (bool, error) {
	v, err := c.do([]byte("DEL"), []byte(key))
	if err != nil {
		return false, err
	}
	n, ok := v.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected type %T", v)
	}
	return n == 1, nil
}

// Exists returns true if key exists.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	v, err := c.do([]byte("EXISTS"), []byte(key))
	if err != nil {
		return false, err
	}
	n, ok := v.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected type %T", v)
	}
	return n == 1, nil
}

// Keys returns keys matching pattern (e.g. "*" for all).
func (c *Client) Keys(ctx context.Context, pattern string) ([][]byte, error) {
	v, err := c.do([]byte("KEYS"), []byte(pattern))
	if err != nil {
		return nil, err
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", v)
	}
	out := make([][]byte, 0, len(arr))
	for _, a := range arr {
		b, ok := a.([]byte)
		if !ok {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

// Ping returns PONG or the optional message.
func (c *Client) Ping(ctx context.Context, msg string) ([]byte, error) {
	if msg == "" {
		v, err := c.do([]byte("PING"))
		if err != nil {
			return nil, err
		}
		if v == nil {
			return []byte("PONG"), nil
		}
		b, ok := v.([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T", v)
		}
		return b, nil
	}
	v, err := c.do([]byte("PING"), []byte(msg))
	if err != nil {
		return nil, err
	}
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", v)
	}
	return b, nil
}

// Do sends a raw command and returns the response (for advanced use).
func (c *Client) Do(ctx context.Context, cmd string, args ...[]byte) (interface{}, error) {
	argv := make([][]byte, 0, 1+len(args))
	argv = append(argv, []byte(cmd))
	argv = append(argv, args...)
	return c.do(argv...)
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
