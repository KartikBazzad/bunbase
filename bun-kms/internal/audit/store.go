package audit

// Store is a placeholder for persistent audit storage (e.g. separate append-only store).
// The current implementation uses Logger writing to a file; this interface allows
// future backends (e.g. HMAC-signed append-only store) without changing callers.
type Store interface {
	Log(Event) error
	Close() error
}

// Ensure Logger implements Store.
var _ Store = (*Logger)(nil)
