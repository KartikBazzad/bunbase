package docdb

// This file previously contained an experimental TransactionBuffer used to
// prototype commit marker handling. The final v0.1 design uses WAL commit
// markers plus recovery-time filtering (see replayWAL) instead, so the
// TransactionBuffer type is no longer needed.
