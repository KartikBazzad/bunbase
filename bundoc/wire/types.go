package wire

// Options mirrors query.Options for wire transport
type Options struct {
	SortField string `json:"sort_field,omitempty"`
	SortDesc  bool   `json:"sort_desc,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Skip      int    `json:"skip,omitempty"`
}

// Request metadata
type RequestMeta struct {
	ProjectID  string `json:"project"`
	Database   string `json:"db"` // defaults to "default"
	Collection string `json:"coll"`
}

// InsertRequest (OpInsert)
type InsertRequest struct {
	RequestMeta
	Document map[string]interface{} `json:"doc"`
}

// FindRequest (OpFind)
type FindRequest struct {
	RequestMeta
	Query   map[string]interface{} `json:"query"`
	Options Options                `json:"opts,omitempty"`
}

// UpdateRequest (OpUpdate)
type UpdateRequest struct {
	RequestMeta
	Filter map[string]interface{} `json:"filter"`
	Update map[string]interface{} `json:"update"`
}

// DeleteRequest (OpDelete)
type DeleteRequest struct {
	RequestMeta
	Filter map[string]interface{} `json:"filter"`
}

// Reply (OpReply or OpError)
type Reply struct {
	Error string                   `json:"error,omitempty"`
	Docs  []map[string]interface{} `json:"docs,omitempty"`
	Count int                      `json:"count,omitempty"`
}

// -- Raft Types --

// LogEntry represents a single replicated command in Raft.
type LogEntry struct {
	Term    uint64 `json:"term"` // Term when entry was received by leader
	Index   uint64 `json:"idx"`  // Monotonic log index
	Command []byte `json:"cmd"`  // Encoded state machine command
}

// RequestVoteRequest (OpRequestVote).
// Invoked by candidates to gather votes.
type RequestVoteRequest struct {
	Term         uint64 `json:"term"`          // Candidate's term
	CandidateID  string `json:"cand_id"`       // Candidate requesting vote
	LastLogIndex uint64 `json:"last_log_idx"`  // Index of candidate's last log entry
	LastLogTerm  uint64 `json:"last_log_term"` // Term of candidate's last log entry
}

// RequestVoteReply
type RequestVoteReply struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"vote"`
}

// AppendEntriesRequest (OpAppendEntries)
type AppendEntriesRequest struct {
	Term         uint64     `json:"term"`
	LeaderID     string     `json:"leader_id"`
	PrevLogIndex uint64     `json:"prev_log_idx"`
	PrevLogTerm  uint64     `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit uint64     `json:"commit_idx"`
}

// AppendEntriesReply
type AppendEntriesReply struct {
	Term    uint64 `json:"term"`
	Success bool   `json:"success"`
}

// -- Authentication Types --

// AuthRequest (OpAuth Client -> Server)
// Step 1: Connect(User) -> Server Challenge matches
// Step 3: ClientProof -> Server Verifies
type AuthRequest struct {
	RequestMeta
	Step     int    `json:"step"` // 1=Connect, 2=Proof
	Username string `json:"username,omitempty"`
	Proof    string `json:"proof,omitempty"`
}

// AuthChallenge (OpAuthReply Server -> Client)
// Step 2: Server sends Salt + Iters
type AuthChallenge struct {
	Salt       string `json:"salt"`
	Iterations int    `json:"iters"`
	ServerKey  string `json:"server_key,omitempty"` // Sent on success to mutually auth?
	SessionID  string `json:"session_id,omitempty"`
}
