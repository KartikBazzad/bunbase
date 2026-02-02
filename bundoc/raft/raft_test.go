package raft

import (
	"fmt"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// MockRPC
type MockRPC struct {
	peers map[string]*Node
}

func (m *MockRPC) SendRequestVote(peer string, args wire.RequestVoteRequest) (wire.RequestVoteReply, error) {
	p, ok := m.peers[peer]
	if !ok {
		return wire.RequestVoteReply{}, fmt.Errorf("peer not found")
	}
	return p.RequestVote(args), nil
}

func (m *MockRPC) SendAppendEntries(peer string, args wire.AppendEntriesRequest) (wire.AppendEntriesReply, error) {
	p, ok := m.peers[peer]
	if !ok {
		return wire.AppendEntriesReply{}, fmt.Errorf("peer not found")
	}
	// Simulate network delay
	// time.Sleep(1 * time.Millisecond)
	return p.AppendEntries(args), nil
}

// MockFSM
type MockFSM struct {
	Applied []string
}

func (m *MockFSM) Apply(cmd []byte) interface{} {
	m.Applied = append(m.Applied, string(cmd))
	return nil
}

func createCluster(t *testing.T, n int) ([]*Node, *MockRPC) {
	peers := make([]string, n)
	for i := 0; i < n; i++ {
		peers[i] = fmt.Sprintf("node%d", i)
	}

	nodes := make([]*Node, n)
	mockRPC := &MockRPC{peers: make(map[string]*Node)}

	for i := 0; i < n; i++ {
		cfg := DefaultConfig(peers[i], peers)
		cfg.ElectionMinMs = 150
		cfg.ElectionMaxMs = 300

		nodes[i] = NewNode(cfg, mockRPC, &MockFSM{})
		mockRPC.peers[peers[i]] = nodes[i]
	}

	return nodes, mockRPC
}

func TestLeaderElection(t *testing.T) {
	nodes, _ := createCluster(t, 3)

	for _, n := range nodes {
		n.Start()
		defer n.Stop()
	}

	// Wait for election
	time.Sleep(500 * time.Millisecond)

	leaders := 0
	for _, n := range nodes {
		n.mu.Lock()
		if n.state == Leader {
			leaders++
		}
		n.mu.Unlock()
	}

	if leaders != 1 {
		t.Errorf("Expected 1 leader, got %d", leaders)
	}
}

func TestLogReplication(t *testing.T) {
	nodes, _ := createCluster(t, 3)

	for _, n := range nodes {
		n.Start()
		defer n.Stop()
	}

	// Wait for leader
	time.Sleep(500 * time.Millisecond)

	var leader *Node
	for _, n := range nodes {
		n.mu.Lock()
		if n.state == Leader {
			leader = n
			n.mu.Unlock()
			break
		}
		n.mu.Unlock()
	}

	if leader == nil {
		t.Fatal("No leader elected")
	}

	// Append log to leader manually (since we don't have public API for this yet)
	// Actually we need a way to propose a command.
	// Assume we have client API or internal method.
	// But `Node` doesn't have `Propose(cmd)` yet?
	// We need to implement `Propose` or similar.

	// Let's implement `Propose` in `node.go` or just modify log directly for test
	// Modifying log directly is racey if node runs.
	// Let's add `Propose` method to Node in the test for now or assume internal access.
	// We are in `raft` package so we can access private fields but `mu` protects them.

	cmd := []byte("cmd1")
	leader.mu.Lock()
	entry := wire.LogEntry{
		Term:    leader.currentTerm,
		Index:   uint64(len(leader.log)), // 0-based index?
		Command: cmd,
	}
	// If log is empty (len=0), first index is 1?
	// In `Node.NewNode`, we didn't add dummy entry.
	// `AppendEntries` logic assumes 0-based.
	// Let's assume index starts at 1.
	// If empty, prevLogIndex=0.
	entry.Index = uint64(len(leader.log) + 1)
	leader.log = append(leader.log, entry)
	leader.mu.Unlock()

	// Wait for heartbeats to propagate
	time.Sleep(200 * time.Millisecond)

	// Check followers
	replicatedCount := 0
	for _, n := range nodes {
		if n == leader {
			continue
		}
		n.mu.Lock()
		if len(n.log) == 1 && string(n.log[0].Command) == "cmd1" {
			replicatedCount++
		}
		n.mu.Unlock()
	}

	if replicatedCount == 0 {
		t.Errorf("Log not replicated to followers")
	}
}
