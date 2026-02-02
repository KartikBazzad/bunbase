package raft

import (
	"log"
	"sync/atomic"

	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// RequestVote handles an incoming vote request from a Candidate.
//
// Logic:
// 1. **Reject** if Candidate's term is older than ours (Reply false).
// 2. **Step Down** if Candidate's term is newer (Update term, become Follower).
// 3. **Vote Check**: Grant vote IF:
//   - We haven't voted for anyone else in this term OR we voted for this Candidate.
//   - AND Candidate's log is at least as up-to-date as ours (Raft Safety Property).
func (n *Node) RequestVote(args wire.RequestVoteRequest) wire.RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := wire.RequestVoteReply{
		Term:        n.currentTerm,
		VoteGranted: false,
	}

	// 1. Reply false if term < currentTerm
	if args.Term < n.currentTerm {
		return reply
	}

	// If RPC request contains a higher term, update currentTerm and convert to follower
	if args.Term > n.currentTerm {
		n.currentTerm = args.Term
		n.state = Follower
		n.votedFor = ""
		n.resetElectionTimer()
	}

	// 2. If votedFor is null or candidateId, and candidate's log is at least as up-to-date as receiver's log, grant vote
	// Check Log Up-to-date
	lastIdx, lastTerm := n.getLastLogInfo()
	isUpToDate := false
	if args.LastLogTerm > lastTerm {
		isUpToDate = true
	} else if args.LastLogTerm == lastTerm && args.LastLogIndex >= lastIdx {
		isUpToDate = true
	}

	if (n.votedFor == "" || n.votedFor == args.CandidateID) && isUpToDate {
		n.votedFor = args.CandidateID
		n.resetElectionTimer() // Granting vote resets timer
		reply.VoteGranted = true
		reply.Term = n.currentTerm // Return updated term
		// log.Printf("[%s] Voted FOR %s in Term %d", n.id, args.CandidateID, n.currentTerm)
	}

	return reply
}

// startElection (continued from node.go)
// Assumes caller holds lock (but we need to release for IO)
// Wait, startElection in node.go held lock.
// We should perform IO asynchronously.
func (n *Node) runElection() {
	n.mu.Lock()
	term := n.currentTerm
	// id := n.id
	peers := n.peers
	n.mu.Unlock()

	var votesReceived int32 = 1 // Vote for self

	for _, peer := range peers {
		if peer == n.id {
			continue
		}

		go func(p string) {
			n.mu.Lock()
			lastIdx, lastTerm := n.getLastLogInfo()
			n.mu.Unlock()

			args := wire.RequestVoteRequest{
				Term:         term,
				CandidateID:  n.id,
				LastLogIndex: lastIdx,
				LastLogTerm:  lastTerm,
			}

			reply, err := n.rpc.SendRequestVote(p, args)
			if err != nil {
				// log.Printf("Failed to RequestVote from %s: %v", p, err)
				return
			}

			n.mu.Lock()
			defer n.mu.Unlock()

			if n.state != Candidate || n.currentTerm != term {
				return // Election obsolete
			}

			if reply.Term > n.currentTerm {
				n.currentTerm = reply.Term
				n.state = Follower
				n.votedFor = ""
				n.resetElectionTimer()
				return
			}

			if reply.VoteGranted {
				votes := atomic.AddInt32(&votesReceived, 1)
				// Check for Majority
				if int(votes) > len(n.peers)/2 {
					n.becomeLeader()
				}
			}
		}(peer)
	}
}

func (n *Node) becomeLeader() {
	if n.state == Leader {
		return
	}
	n.state = Leader
	log.Printf("[%s] Became LEADER Term %d", n.id, n.currentTerm)

	// Stop election timer
	if n.electionTimer != nil {
		n.electionTimer.Stop()
	}

	// Initialize Leader State
	n.nextIndex = make(map[string]uint64)
	n.matchIndex = make(map[string]uint64)
	lastIdx, _ := n.getLastLogInfo()

	for _, p := range n.peers {
		n.nextIndex[p] = lastIdx + 1
		n.matchIndex[p] = 0
	}

	// Start Heartbeat
	n.startHeartbeat()
}

func (n *Node) getLastLogInfo() (uint64, uint64) {
	if len(n.log) == 0 {
		return 0, 0
	}
	last := n.log[len(n.log)-1]
	return last.Index, last.Term
}
