package wal

// Recover reads all WAL records from all segment files for replay after a crash.
// No transaction filtering: each record (Set/Del/Expire) is applied in order.
func Recover(w *WAL) ([]*Record, error) {
	return w.ReadAllRecords()
}
