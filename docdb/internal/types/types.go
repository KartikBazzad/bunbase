package types

import "time"

type OperationType byte

const (
	OpCreate OperationType = iota + 1
	OpRead
	OpUpdate
	OpDelete
	OpCommit
	OpCheckpoint
	OpPatch
	OpCreateCollection
	OpDeleteCollection
)

type Status byte

const (
	StatusOK Status = iota
	StatusError
	StatusNotFound
	StatusConflict
	StatusMemoryLimit
)

type Document struct {
	ID      uint64
	Payload []byte
}

type DocumentVersion struct {
	ID          uint64
	CreatedTxID uint64
	DeletedTxID *uint64
	Offset      uint64
	Length      uint32
}

type Transaction struct {
	ID           uint64
	SnapshotTxID uint64
	StartedAt    time.Time
}

type DBStatus byte

const (
	DBActive DBStatus = iota + 1
	DBDeleted
)

type CatalogEntry struct {
	DBID      uint64
	DBName    string
	CreatedAt time.Time
	Status    DBStatus
}

type WALRecord struct {
	Length     uint64
	LSN        uint64 // v0.4: partition-local LSN (0 for v0.1/v0.2)
	TxID       uint64
	DBID       uint64
	Collection string // Collection name (empty for v0.1 records, defaults to "_default")
	OpType     OperationType
	DocID      uint64
	PayloadLen uint32
	Payload    []byte
	CRC        uint32
}

type PatchOperation struct {
	Op    string      // "set", "delete", "insert"
	Path  string      // JSON Pointer-like path (e.g., "/name", "/address/city")
	Value interface{} // JSON value (required for set/insert, nil for delete)
}

type CollectionMetadata struct {
	Name      string
	CreatedAt time.Time
	DocCount  uint64
}

type DBInfo struct {
	Name       string
	ID         uint64
	CreatedAt  time.Time
	WALSize    uint64
	MemoryUsed uint64
	DocsLive   uint64
}

type Stats struct {
	TotalDBs       int
	ActiveDBs      int
	TotalTxns      uint64
	TxnsCommitted  uint64
	WALSize        uint64
	MemoryUsed     uint64
	MemoryCapacity uint64
	DocsLive       uint64
	DocsTombstoned uint64
	LastCompaction time.Time
}
