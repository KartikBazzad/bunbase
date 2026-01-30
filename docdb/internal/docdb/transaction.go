package docdb

import (
	"errors"
	"sync"

	"github.com/kartikbazzad/docdb/internal/types"
)

var (
	ErrTxNotFound          = errors.New("transaction not found")
	ErrTxAlreadyCommitted  = errors.New("transaction already committed")
	ErrTxAlreadyRolledBack = errors.New("transaction already rolled back")
	ErrSerializationFailure = errors.New("serialization failure: conflict with concurrent transaction")
)

type TxState int

const (
	TxOpen TxState = iota
	TxCommitted
	TxRolledBack
)

type Tx struct {
	ID           uint64
	SnapshotTxID uint64
	Operations   []*types.WALRecord
	state        TxState
	// readSet records (collection, docID) read via ReadInTx for SSI conflict detection.
	// Key format: "collection:docID". Nil until first ReadInTx; initialized lazily or in Begin.
	readSet map[string]struct{}
}

type TransactionManager struct {
	mu   sync.RWMutex
	txs  map[uint64]*Tx
	mvcc *MVCC
}

func NewTransactionManager(mvcc *MVCC) *TransactionManager {
	return &TransactionManager{
		txs:  make(map[uint64]*Tx),
		mvcc: mvcc,
	}
}

func (tm *TransactionManager) Begin() *Tx {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	txID := tm.mvcc.NextTxID()
	snapshotTxID := tm.mvcc.CurrentSnapshot()

	tx := &Tx{
		ID:           txID,
		SnapshotTxID: snapshotTxID,
		Operations:   make([]*types.WALRecord, 0),
		state:        TxOpen,
		readSet:      make(map[string]struct{}),
	}

	tm.txs[txID] = tx
	return tx
}

func (tm *TransactionManager) AddOp(tx *Tx, dbID uint64, collection string, opType types.OperationType, docID uint64, payload []byte) error {
	if tx.state != TxOpen {
		return ErrTxAlreadyCommitted
	}

	if collection == "" {
		collection = DefaultCollection
	}

	record := &types.WALRecord{
		TxID:       tx.ID,
		DBID:       dbID,
		Collection: collection,
		OpType:     opType,
		DocID:      docID,
		PayloadLen: uint32(len(payload)),
		Payload:    make([]byte, len(payload)),
	}
	copy(record.Payload, payload)

	tx.Operations = append(tx.Operations, record)
	return nil
}

func (tm *TransactionManager) Commit(tx *Tx) ([]*types.WALRecord, error) {
	if tx.state != TxOpen {
		return nil, ErrTxAlreadyCommitted
	}

	tx.state = TxCommitted
	return tx.Operations, nil
}

func (tm *TransactionManager) Rollback(tx *Tx) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tx.state != TxOpen {
		return ErrTxAlreadyRolledBack
	}

	tx.state = TxRolledBack
	delete(tm.txs, tx.ID)
	return nil
}

func (tm *TransactionManager) Get(txID uint64) (*Tx, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tx, exists := tm.txs[txID]
	if !exists {
		return nil, ErrTxNotFound
	}
	return tx, nil
}
