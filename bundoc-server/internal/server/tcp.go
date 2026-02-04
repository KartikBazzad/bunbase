package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/kartikbazzad/bunbase/bundoc"
	"github.com/kartikbazzad/bunbase/bundoc-server/internal/manager"
	"github.com/kartikbazzad/bunbase/bundoc/mvcc"
	"github.com/kartikbazzad/bunbase/bundoc/raft"
	"github.com/kartikbazzad/bunbase/bundoc/security"
	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// TCPServer handles raw TCP connections using the Wire Protocol
type TCPServer struct {
	addr      string
	manager   *manager.InstanceManager
	ln        net.Listener
	wg        sync.WaitGroup
	quit      chan struct{}
	raftNode  *raft.Node
	tlsConfig *tls.Config // Optional TLS Config
}

func NewTCPServer(addr string, mgr *manager.InstanceManager, tlsCfg *tls.Config) *TCPServer {
	return &TCPServer{
		addr:      addr,
		manager:   mgr,
		quit:      make(chan struct{}),
		tlsConfig: tlsCfg,
	}
}

func (s *TCPServer) Start() error {
	var ln net.Listener
	var err error

	if s.tlsConfig != nil {
		ln, err = tls.Listen("tcp", s.addr, s.tlsConfig)
		log.Printf("🔒 Bundoc TCP Server (TLS) listening on %s", s.addr)
	} else {
		ln, err = net.Listen("tcp", s.addr)
		log.Printf("🚀 Bundoc TCP Server (Plain) listening on %s", s.addr)
	}

	if err != nil {
		return err
	}
	s.ln = ln

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

// Session holds connection state
type Session struct {
	User *security.User
	ID   string
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	session := &Session{}

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
			s.handleInsert(conn, req, session)

		case wire.OpFind:
			var req wire.FindRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleFind(conn, req, session)

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

		case wire.OpAuth:
			var req wire.AuthRequest
			if err := wire.ReadBody(conn, header.Length, &req); err != nil {
				s.sendError(conn, "Invalid Body: "+err.Error())
				continue
			}
			s.handleAuth(conn, req, session)

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

func (s *TCPServer) handleAuth(w io.Writer, req wire.AuthRequest, sess *Session) {
	// 1. Acquire DB
	db, release, err := s.manager.Acquire(req.ProjectID)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}
	defer release()

	if db.Security == nil {
		s.sendError(w, "Security not enabled on server")
		return
	}

	// 2. Handle Steps
	// Step 1: Initial Handshake (Client sends Username)
	if req.Step == 1 {
		creds, err := db.Security.GetSCRAMCredentials(req.Username)
		if err != nil {
			// Don't reveal user existence? generic error?
			// For internal DB, specific error helps debugging.
			s.sendError(w, "Auth Failed: "+err.Error())
			return
		}

		reply := wire.AuthChallenge{
			Salt:       creds.Salt,
			Iterations: creds.Iterations,
		}
		wire.WriteMessage(w, wire.OpAuthReply, reply)
		return
	}

	// Step 2: Verification (Client sends Proof)
	if req.Step == 2 {
		creds, err := db.Security.GetSCRAMCredentials(req.Username)
		if err != nil {
			s.sendError(w, "Auth Failed: User not found")
			return
		}

		// Simplified Verification for MVP:
		// TODO: Add Nonces to AuthRequest/Challenge for replay protection.
		authMessage := "bundoc-auth"

		if security.VerifyClientProof(creds.StoredKey, authMessage, req.Proof) {
			// Success! Load User into Session
			user, err := db.Security.GetUser(req.Username)
			if err == nil {
				sess.User = user
				sess.ID = "sess-" + req.Username
			}

			// Audit Success
			db.Audit.Log(security.EventLoginSuccess, req.Username, "", nil)

			// Return ServerKey for mutual auth?
			wire.WriteMessage(w, wire.OpAuthReply, wire.AuthChallenge{
				ServerKey: creds.ServerKey,
				SessionID: sess.ID,
			})
		} else {
			// Audit Failure
			db.Audit.Log(security.EventLoginFailure, req.Username, "", map[string]interface{}{"reason": "invalid_proof"})
			s.sendError(w, "Authentication Failed: Invalid Proof")
		}
		return
	}

	s.sendError(w, "Invalid Auth Step")
}

func (s *TCPServer) handleInsert(w io.Writer, req wire.InsertRequest, sess *Session) {
	// RBAC Check
	if sess.User == nil {
		s.sendError(w, "Unauthorized: Please login first")
		return
	}
	// Check Permission on this Database
	// We map req.Database to permission scope? Or default scope.
	targetDB := req.Database
	if targetDB == "" {
		targetDB = "default" // Or "" global
	}
	// Actually, just verify PermWrite anywhere for now?
	// Or check User.HasPermission("", PermWrite) meaning global write?
	// Let's assume global write requirement for any insert for MVP.
	if !sess.User.HasPermission("", security.PermWrite) && !sess.User.HasPermission(targetDB, security.PermWrite) {
		// Log Audit (Need DB access, but DB is acquired later.
		// Wait, we haven't acquired DB yet. We need 'db' to access 'Audit'.
		// This means we need to look up Audit Logger from Manager or a global place?
		// Or acquire DB just to log?
		// Actually, manager.Acquire returns *bundoc.Database.
		// So we must acquire DB first to log to its audit log.
		// But here we check permissions BEFORE acquiring DB (for speed/DDoS?)
		// If we block before acquire, we can't write to DB's audit log.
		// Solution: Acquire DB earlier or allow logging via Manager?
		// For now, let's just acquire DB earlier. It is cheap (map lookup).
	}

	// Acquire DB
	db, release, err := s.manager.Acquire(req.ProjectID)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}
	defer release()

	if !sess.User.HasPermission("", security.PermWrite) && !sess.User.HasPermission(targetDB, security.PermWrite) {
		db.Audit.Log(security.EventAccessDenied, sess.User.Username, "", map[string]interface{}{
			"action": "insert",
			"db":     targetDB,
			"col":    req.Collection,
		})
		s.sendError(w, fmt.Sprintf("Forbidden: User %s missing write permission", sess.User.Username))
		return
	}

	// get Collection...
	col, err := db.GetCollection(req.Collection)
	if err != nil {
		// Create requires PermAdmin or PermWrite? Usually Write is enough for implicit create in Mongo,
		// but secure systems might require Admin. Let's allow Write to create for usability.
		col, err = db.CreateCollection(req.Collection)
		if err != nil {
			s.sendError(w, err.Error())
			return
		}
	}

	// Execute Insert
	txn, _ := db.BeginTransaction(mvcc.ReadCommitted)
	if err := col.Insert(nil, txn, req.Document); err != nil {
		db.RollbackTransaction(txn)
		s.sendError(w, err.Error())
		return
	}
	db.CommitTransaction(txn)

	s.sendReply(w, wire.Reply{Count: 1})
}

func (s *TCPServer) handleFind(w io.Writer, req wire.FindRequest, sess *Session) {
	// RBAC Check
	if sess.User == nil {
		s.sendError(w, "Unauthorized: Please login first")
		return
	}
	// Check Read Permission
	targetDB := req.Database
	if targetDB == "" {
		targetDB = "default"
	}

	// Acquire DB
	db, release, err := s.manager.Acquire(req.ProjectID)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}
	defer release()

	if !sess.User.HasPermission("", security.PermRead) && !sess.User.HasPermission(targetDB, security.PermRead) {
		db.Audit.Log(security.EventAccessDenied, sess.User.Username, "", map[string]interface{}{
			"action": "find",
			"db":     targetDB,
			"col":    req.Collection,
		})
		s.sendError(w, fmt.Sprintf("Forbidden: User %s missing read permission", sess.User.Username))
		return
	}

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
	docs, err := col.FindQuery(nil, txn, req.Query, qOpts)
	if err != nil {
		s.sendError(w, err.Error())
		return
	}

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
