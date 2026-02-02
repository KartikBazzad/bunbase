package raft

import (
	"fmt"
	"net"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// TCPTransport implements RPCClient using the Wire Protocol over TCP
type TCPTransport struct {
	Timeout time.Duration
}

func NewTCPTransport() *TCPTransport {
	return &TCPTransport{
		Timeout: 2 * time.Second, // Fast timeout for internal RPCs
	}
}

func (t *TCPTransport) SendRequestVote(peer string, args wire.RequestVoteRequest) (wire.RequestVoteReply, error) {
	conn, err := net.DialTimeout("tcp", peer, t.Timeout)
	if err != nil {
		return wire.RequestVoteReply{}, err
	}
	defer conn.Close()

	// Send Request
	if err := wire.WriteMessage(conn, wire.OpRequestVote, args); err != nil {
		return wire.RequestVoteReply{}, err
	}

	// Read Reply
	header, err := wire.ReadHeader(conn)
	if err != nil {
		return wire.RequestVoteReply{}, err
	}

	// Expect OpReply? Or OpRequestVoteReply?
	// Note: protocol.go: OpReply is generic.
	// We should probably encoding the specific reply struct in body.
	// If the server sends OpReply, we read RequestVoteReply from body?
	// Or maybe use specific OpCodes for replies?
	// Standard simple RPC: Request OpCode -> Response is generically OpReply with body being the specific struct?
	// Or Request OpCode -> Response OpCode?
	// Let's assume server sends back OpReply with JSON body matching RequestVoteReply.

	if header.OpCode == wire.OpError {
		// handle error
		var errReply wire.Reply
		wire.ReadBody(conn, header.Length, &errReply)
		return wire.RequestVoteReply{}, fmt.Errorf("rpc error: %s", errReply.Error)
	}

	var reply wire.RequestVoteReply
	if err := wire.ReadBody(conn, header.Length, &reply); err != nil {
		return wire.RequestVoteReply{}, err
	}

	return reply, nil
}

func (t *TCPTransport) SendAppendEntries(peer string, args wire.AppendEntriesRequest) (wire.AppendEntriesReply, error) {
	conn, err := net.DialTimeout("tcp", peer, t.Timeout)
	if err != nil {
		return wire.AppendEntriesReply{}, err
	}
	defer conn.Close()

	if err := wire.WriteMessage(conn, wire.OpAppendEntries, args); err != nil {
		return wire.AppendEntriesReply{}, err
	}

	header, err := wire.ReadHeader(conn)
	if err != nil {
		return wire.AppendEntriesReply{}, err
	}

	if header.OpCode == wire.OpError {
		var errReply wire.Reply
		wire.ReadBody(conn, header.Length, &errReply)
		return wire.AppendEntriesReply{}, fmt.Errorf("rpc error: %s", errReply.Error)
	}

	var reply wire.AppendEntriesReply
	if err := wire.ReadBody(conn, header.Length, &reply); err != nil {
		return wire.AppendEntriesReply{}, err
	}

	return reply, nil
}
