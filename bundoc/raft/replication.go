package raft

import (
	"log"
	"time"

	"github.com/kartikbazzad/bunbase/bundoc/wire"
)

// AppendEntries handles log replication requests from the Leader.
//
// Logic:
// 1. **Term Check**: Reject if Leader's term is older than ours.
// 2. **Step Down**: If Leader's term is newer, update ours and become Follower.
// 3. **Consistency Check**: Verify that `PrevLogIndex` and `PrevLogTerm` match our log.
//   - If not, return valid=false (Leader will decrement nextIndex and retry).
//     4. **Conflict Resolution**: If a new entry conflicts with an existing one (same index, different term),
//     truncate the existing entry and all following it.
//     5. **Append**: Add new entries to the log.
//     6. **Commit Update**: Update `commitIndex` based on Leader's commit index.
func (n *Node) AppendEntries(args wire.AppendEntriesRequest) wire.AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := wire.AppendEntriesReply{
		Term:    n.currentTerm,
		Success: false,
	}

	// 1. Reply false if term < currentTerm
	if args.Term < n.currentTerm {
		return reply
	}

	// Update Term if needed
	n.resetElectionTimer() // Recognized valid leader
	if args.Term > n.currentTerm {
		n.currentTerm = args.Term
		n.state = Follower
		n.votedFor = ""
	}
	n.leaderID = args.LeaderID

	// 2. Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm
	if args.PrevLogIndex > 0 {
		// Log index check.
		// Note: We use 0-indexed slice but Raft usually 1-indexed.
		// Let's assume LogEntry.Index is the truth.
		// We need to find if we have an entry with that Index and Term.

		// Optimization: Check last log entry
		lastIdx, _ := n.getLastLogInfo()
		if lastIdx < args.PrevLogIndex {
			return reply // Missing previous entry
		}

		// Use binary search or direct index if consistent
		// Finding entry with Index == args.PrevLogIndex
		// Assuming contiguous: entry at index [idx] if log[0].Index=1?
		// We can scan or use map? Array scan is fine for now or helper.
		entry, found := n.getLogEntry(args.PrevLogIndex)
		if !found || entry.Term != args.PrevLogTerm {
			return reply // Term mismatch
		}
	}

	// 3. If an existing entry conflicts with a new one (same index but different terms),
	//    delete the existing entry and all that follow it.
	// 4. Append any new entries not already in the log.
	for _, newEntry := range args.Entries {
		existing, found := n.getLogEntry(newEntry.Index)
		if found {
			if existing.Term != newEntry.Term {
				// Conflict: Delete inclusive
				n.truncateLog(newEntry.Index)
				n.log = append(n.log, newEntry)
			}
		} else {
			// Append
			n.log = append(n.log, newEntry)
		}
	}

	// 5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > n.commitIndex {
		lastIdx, _ := n.getLastLogInfo()
		if args.LeaderCommit < lastIdx {
			n.commitIndex = args.LeaderCommit
		} else {
			n.commitIndex = lastIdx
		}
		// Signal apply
		n.applyLogs()
	}

	reply.Success = true
	reply.Term = n.currentTerm
	return reply
}

func (n *Node) startHeartbeat() {
	if n.heartbeatTimer != nil {
		n.heartbeatTimer.Stop()
	}
	n.heartbeatTimer = time.NewTicker(time.Duration(n.config.HeartbeatMs) * time.Millisecond)

	go func() {
		for {
			select {
			case <-n.heartbeatTimer.C:
				n.mu.Lock()
				if n.state != Leader {
					n.heartbeatTimer.Stop()
					n.mu.Unlock()
					return
				}
				term := n.currentTerm
				// leaderID := n.id
				n.mu.Unlock()

				n.broadcastAppendEntries(term)

			case <-n.stopCh:
				return
			}
		}
	}()
}

func (n *Node) broadcastAppendEntries(term uint64) {
	n.mu.Lock()
	peers := n.peers
	n.mu.Unlock()

	for _, peer := range peers {
		if peer == n.id {
			continue
		}

		go func(p string) {
			n.mu.Lock()
			nextIdx := n.nextIndex[p]
			prevLogIndex := nextIdx - 1
			prevLogTerm := uint64(0)

			if prevLogIndex > 0 {
				entry, found := n.getLogEntry(prevLogIndex)
				if found {
					prevLogTerm = entry.Term
				}
			}

			// Collect entries to send (from nextIdx onwards)
			var entries []wire.LogEntry
			for _, entry := range n.log {
				if entry.Index >= nextIdx {
					entries = append(entries, entry)
				}
			}

			// If too many, limit? (Batching)

			leaderCommit := n.commitIndex
			n.mu.Unlock()

			args := wire.AppendEntriesRequest{
				Term:         term,
				LeaderID:     n.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: leaderCommit,
			}

			reply, err := n.rpc.SendAppendEntries(p, args)
			if err != nil {
				return
			}

			n.mu.Lock()
			defer n.mu.Unlock()

			if n.state != Leader || n.currentTerm != term {
				return
			}

			if reply.Term > n.currentTerm {
				n.currentTerm = reply.Term
				n.state = Follower
				n.votedFor = ""
				n.resetElectionTimer() // Also stop heartbeat effectively?
				return
			}

			if reply.Success {
				// Update nextIndex and matchIndex
				if len(entries) > 0 {
					lastEntry := entries[len(entries)-1]
					n.matchIndex[p] = lastEntry.Index
					n.nextIndex[p] = lastEntry.Index + 1

					// Update commitIndex?
					// If there exists an N such that N > commitIndex, a majority of matchIndex[i] >= N,
					// and log[N].term == currentTerm: set commitIndex = N
					n.updateCommitIndex()
				}
			} else {
				// Failed consistency check, decrement nextIndex
				if n.nextIndex[p] > 1 {
					n.nextIndex[p]--
				}
			}
		}(peer)
	}
}

// Helpers

func (n *Node) getLogEntry(index uint64) (wire.LogEntry, bool) {
	// Linear scan for now, assumes no holes
	for _, entry := range n.log {
		if entry.Index == index {
			return entry, true
		}
	}
	return wire.LogEntry{}, false
}

func (n *Node) truncateLog(index uint64) {
	// Keep entries with Index < index
	var newLog []wire.LogEntry
	for _, entry := range n.log {
		if entry.Index < index {
			newLog = append(newLog, entry)
		}
	}
	n.log = newLog
}

func (n *Node) applyLogs() {
	// Apply entries from lastApplied+1 to commitIndex
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		entry, found := n.getLogEntry(n.lastApplied)
		if found {
			// Apply to State Machine
			if n.fsm != nil {
				// We might need to ensure apply is serial and potentially release lock?
				// Typically FSM apply is done via channel or callback.
				// n.fsm.Apply(entry.Command)
				// Or use n.applyCh if using channel
			}
			log.Printf("[%s] Applied Log Index %d (Term %d)", n.id, n.lastApplied, entry.Term)
		}
	}
}

func (n *Node) updateCommitIndex() {
	// Find N
	// Very naive: iterate backwards from last log index
	lastIdx, _ := n.getLastLogInfo()
	for N := lastIdx; N > n.commitIndex; N-- {
		entry, found := n.getLogEntry(N)
		if !found || entry.Term != n.currentTerm {
			continue
		}

		count := 1 // Self
		for _, peer := range n.peers {
			if peer == n.id {
				continue
			}
			if n.matchIndex[peer] >= N {
				count++
			}
		}

		if count > len(n.peers)/2 {
			n.commitIndex = N
			n.applyLogs()
			break
		}
	}
}
