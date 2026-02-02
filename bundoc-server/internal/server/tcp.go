package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/raft"
	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// TCPServer handles raw TCP connections using the Wire Protocol
type TCPServer struct {
	addr     string
	manager  *manager.InstanceManager
	ln       net.Listener
	wg       sync.WaitGroup
	quit     chan struct{}
	raftNode *raft.Node
}

func NewTCPServer(addr string, mgr *manager.InstanceManager) *TCPServer {
	return &TCPServer{
		addr:    addr,
		manager: mgr,
		quit:    make(chan struct{}),
	}
}

func (s *TCPServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln
	log.Printf("🚀 Bundoc TCP Server listening on %s", s.addr)

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

func (s *TCPServer) Stop() error {
	close(s.quit)
	if s.ln != nil {
		s.ln.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *TCPServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return // Normal shutdown
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// 1. Read Header
		header, err := wire.ReadHeader(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("ReadHeader error: %v", err)
			}
			return
		}

		// 2. Process based on OpCode
		switch header.OpCode {
		case wire.OpInsert:
			var req wire.InsertRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleInsert(conn, req)

		case wire.OpFind:
			var req wire.FindRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleFind(conn, req)

		case wire.OpRequestVote:
			var req wire.RequestVoteRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleRequestVote(conn, req)

		case wire.OpAppendEntries:
			var req wire.AppendEntriesRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleAppendEntries(conn, req)

		default:
			// Consume body if unknown op
			io.CopyN(io.Discard, conn, int64(header.Length))
			s.sendError(conn, fmt.Sprintf("Unknown OpCode: %d", header.OpCode))
		}
	}
}

func (s *TCPServer) SetRaftNode(node *raft.Node) {
	s.raftNode = node
}

func (s *TCPServer) sendError(w io.Writer, msg string) {
	reply := wire.Reply{Error: msg}
	wire.WriteMessage(w, wire.OpError, reply)
}

func (s *TCPServer) sendReply(w io.Writer, reply wire.Reply) {
	wire.WriteMessage(w, wire.OpReply, reply)
}

// -- Handlers --

func (s *TCPServer) handleInsert(w io.Writer, req wire.InsertRequest) {
	// Acquire DB
	db, release, err := s.manager.Acquire(req.ProjectID)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}
	defer release()

	// Get Collection
	col, err := db.GetCollection(req.Collection)
	if err != nil {
		// Create if not exists
		col, err = db.CreateCollection(req.Collection)
		if err != nil {
			s.sendError(w, err.Error())
			return
		}
	}

	// Execute Insert
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := col.Insert(txn, req.Document); err != nil {
		db.RollbackTransaction(txn)
		s.sendError(w, err.Error())
		return
	}
	db.CommitTransaction(txn)

	s.sendReply(w, wire.Reply{Count: 1})
}

func (s *TCPServer) handleFind(w io.Writer, req wire.FindRequest) {
	// Acquire DB
	db, release, err := s.manager.Acquire(req.ProjectID)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}
	defer release()

	col, err := db.GetCollection(req.Collection)
	if err != nil {
		s.sendError(w, "Collection not found")
		return
	}

	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	defer db.RollbackTransaction(txn) // Read-only

	// Convert Wire Options to Bundoc QueryOptions
	qOpts := bundoc.QueryOptions{
		SortField: req.Options.SortField,
		SortDesc:  req.Options.SortDesc,
		Limit:     req.Options.Limit,
		Skip:      req.Options.Skip,
	}

	// Call FindQuery
	docs, err := col.FindQuery(txn, req.Query, qOpts)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}

	// Convert []storage.Document (map alias) to []map[string]interface{}
	// Since storage.Document is just map[string]interface{}, simple cast might work in loop
	// but direct assignment []storage.Document to []map... fails in Go.

	replyDocs := make([]map[string]interface{}, len(docs))
	for i, d := range docs {
		replyDocs[i] = d
	}

	s.sendReply(w, wire.Reply{Docs: replyDocs, Count: len(docs)})
}

func (s *TCPServer) handleRequestVote(w io.Writer, req wire.RequestVoteRequest) {
	if s.raftNode == nil {
		s.sendError(w, "Raft not enabled")
		return
	}
	reply := s.raftNode.RequestVote(req)
	wire.WriteMessage(w, wire.OpReply, reply)
}

func (s *TCPServer) handleAppendEntries(w io.Writer, req wire.AppendEntriesRequest) {
	if s.raftNode == nil {
		s.sendError(w, "Raft not enabled")
		return
	}
	reply := s.raftNode.AppendEntries(req)
	wire.WriteMessage(w, wire.OpReply, reply)
}
