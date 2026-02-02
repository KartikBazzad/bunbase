// Package raft implements the Raft consensus algorithm for distributed consistency.
//
// It manages:
// - **Leader Election**: Selecting a cluster leader.
// - **Log Replication**: Ensuring all nodes match the leader's log.
// - **Safety**: Guaranteeing committed entries are never lost.
package raft

import (
	"math/rand"
	"sync"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// State represents the current role of the Raft node.
type State int

const (
	Follower  State = iota // Passive, responds to requests
	Candidate              // Active, seeking votes for leadership
	Leader                 // Active, manages replication
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	}
	return "Unknown"
}

// Config holds configuration parameters for a Raft Node.
type Config struct {
	ID            string   // Unique ID of this node
	Peers         []string // Addresses of peer nodes
	StoragePath   string   // Path for persistent storage
	ElectionMinMs int      // Min election timeout duration
	ElectionMaxMs int      // Max election timeout duration
	HeartbeatMs   int      // Interval between heartbeats
}

func DefaultConfig(id string, peers []string) *Config {
	return &Config{
		ID:            id,
		Peers:         peers,
		StoragePath:   "./raft-data",
		ElectionMinMs: 150,
		ElectionMaxMs: 300,
		HeartbeatMs:   50,
	}
}

// RPCClient defines the interface for communicating with peers.
type RPCClient interface {
	SendRequestVote(peer string, args wire.RequestVoteRequest) (wire.RequestVoteReply, error)
	SendAppendEntries(peer string, args wire.AppendEntriesRequest) (wire.AppendEntriesReply, error)
}

// StateMachine defines the interface for the underlying application (e.g., Bundoc DB).
// Committed log entries are applied to the StateMachine.
type StateMachine interface {
	Apply(cmd []byte) interface{}
}

// Node represents a single participant in the Raft cluster.
// It manages the consensus state, log replication, and interactions with the StateMachine.
type Node struct {
	mu sync.Mutex

	// Persistent State (must survive restarts)
	currentTerm uint64
	votedFor    string
	log         []wire.LogEntry

	// Volatile State
	commitIndex uint64 // Index of highest log entry known to be committed
	lastApplied uint64 // Index of highest log entry applied to StateMachine
	state       State
	leaderID    string

	// Volatile State (Leader only)
	nextIndex  map[string]uint64 // For each peer, index of next log entry to send
	matchIndex map[string]uint64 // For each peer, index of highest log entry known to be replicated

	// Config
	id     string
	peers  []string
	config *Config

	// Dependencies
	rpc Interface
	fsm StateMachine

	// Timers
	electionTimer  *time.Timer
	heartbeatTimer *time.Ticker

	// Channels
	applyCh chan wire.LogEntry
	stopCh  chan struct{}
}

// Interface wraps RPCClient to avoid name collision in struct?
// Just use RPCClient
type Interface = RPCClient

// NewNode creates a new Raft node
func NewNode(cfg *Config, rpc RPCClient, fsm StateMachine) *Node {
	n := &Node{
		id:          cfg.ID,
		peers:       cfg.Peers,
		config:      cfg,
		rpc:         rpc,
		fsm:         fsm,
		state:       Follower,
		currentTerm: 0,
		votedFor:    "",
		log:         make([]wire.LogEntry, 0), // Index 0 is often dummy? or 0-indexed?
		// Raft log is usually 1-indexed. Let's use 0-indexed slice but logical 1-based logic or 0-based.
		// Standard Raft uses 1-based index but 0 is valid "no log" marker.
		// Let's assume slice index matches logical index for simplicity if possible, or mapping.
		// Usually a dummy entry at index 0 makes math easier.
		nextIndex:  make(map[string]uint64),
		matchIndex: make(map[string]uint64),
		stopCh:     make(chan struct{}),
	}

	// Initialize with dummy entry so index 1 is first real entry?
	// n.log = append(n.log, wire.LogEntry{Term: 0, Index: 0})

	return n
}

func (n *Node) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Start Election Timer
	n.resetElectionTimer()

	go n.runLoop()
}

func (n *Node) Stop() {
	close(n.stopCh)
}

func (n *Node) runLoop() {
	// Main event loop could handle ticker events here or relying on Timer callbacks spawning goroutines.
	// Often cleaner to have a loop selecting on channels.
	// For simplicity, let's use time.AfterFunc or similar for timers that send to a channel.
}

func (n *Node) resetElectionTimer() {
	if n.electionTimer != nil {
		n.electionTimer.Stop()
	}
	duration := time.Duration(n.config.ElectionMinMs+rand.Intn(n.config.ElectionMaxMs-n.config.ElectionMinMs)) * time.Millisecond
	n.electionTimer = time.AfterFunc(duration, func() {
		n.startElection()
	})
}

func (n *Node) startElection() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state == Leader {
		return
	}

	// Transition to Candidate
	n.state = Candidate
	n.currentTerm++
	n.votedFor = n.id

	// Reset timer
	n.resetElectionTimer()

	// Send RequestVote to all peers
	go n.runElection()
}
