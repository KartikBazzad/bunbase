package types

import "time"

type OperationType byte

const (
	OpCreate OperationType = iota + 1
	OpRead
	OpUpdate
	OpDelete
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
	FilePaths []string
	CreatedAt time.Time
	Status    DBStatus
}

type WALRecord struct {
	Length     uint64
	TxID       uint64
	DBID       uint64
	OpType     OperationType
	DocID      uint64
	PayloadLen uint32
	Payload    []byte
	CRC        uint32
}

type Stats struct {
	TotalDBs       int
	ActiveDBs      int
	TotalTxns      uint64
	WALSize        uint64
	MemoryUsed     uint64
	MemoryCapacity uint64
}
