package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// Client represents a connection to Bundoc Server
type Client struct {
	addr string
	conn net.Conn
	mu   sync.Mutex
}

// Connect connects to the Bundoc Server
func Connect(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		addr: addr,
		conn: conn,
	}, nil
}

// Close closes the connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// Database returns a handle to a database (logical grouping)
func (c *Client) Database(name string) *Database {
	return &Database{
		client: c,
		name:   name,
	}
}

// Database handle
type Database struct {
	client *Client
	name   string
}

// Collection returns a handle to a collection
func (db *Database) Collection(name string) *Collection {
	return &Collection{
		db:   db,
		name: name,
	}
}

// Collection handle
type Collection struct {
	db   *Database
	name string
}

// Insert inserts a document
func (c *Collection) Insert(projectID string, doc map[string]interface{}) error {
	req := wire.InsertRequest{
		RequestMeta: wire.RequestMeta{
			ProjectID:  projectID,
			Database:   c.db.name,
			Collection: c.name,
		},
		Document: doc,
	}

	c.db.client.mu.Lock()
	defer c.db.client.mu.Unlock()

	// Send Request
	if err := wire.WriteMessage(c.db.client.conn, wire.OpInsert, req); err != nil {
		return err
	}

	// Read Reply
	replyHeader, err := wire.ReadHeader(c.db.client.conn)
	if err != nil {
		return err
	}

	if replyHeader.OpCode == wire.OpError {
		var reply wire.Reply
		if err := wire.ReadBody(c.db.client.conn, replyHeader.Length, &reply); err != nil {
			return err
		}
		return fmt.Errorf("server error: %s", reply.Error)
	}

	// Consume success reply body
	var successReply wire.Reply
	if err := wire.ReadBody(c.db.client.conn, replyHeader.Length, &successReply); err != nil {
		return err
	}

	return nil
}

// FindQuery executes a query
func (c *Collection) FindQuery(projectID string, query map[string]interface{}, opts ...bundoc.QueryOptions) ([]map[string]interface{}, error) {
	wireOpts := wire.Options{}
	if len(opts) > 0 {
		wireOpts.SortField = opts[0].SortField
		wireOpts.SortDesc = opts[0].SortDesc
		wireOpts.Limit = opts[0].Limit
		wireOpts.Skip = opts[0].Skip
	}

	req := wire.FindRequest{
		RequestMeta: wire.RequestMeta{
			ProjectID:  projectID,
			Database:   c.db.name,
			Collection: c.name,
		},
		Query:   query,
		Options: wireOpts,
	}

	c.db.client.mu.Lock()
	defer c.db.client.mu.Unlock()

	if err := wire.WriteMessage(c.db.client.conn, wire.OpFind, req); err != nil {
		return nil, err
	}

	replyHeader, err := wire.ReadHeader(c.db.client.conn)
	if err != nil {
		return nil, err
	}

	var reply wire.Reply
	if err := wire.ReadBody(c.db.client.conn, replyHeader.Length, &reply); err != nil {
		return nil, err
	}

	if replyHeader.OpCode == wire.OpError {
		return nil, fmt.Errorf("server error: %s", reply.Error)
	}

	return reply.Docs, nil
}
