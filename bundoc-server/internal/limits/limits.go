package limits

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Config holds Bundoc resource limits (0 = unlimited).
type Config struct {
	MaxConnectionsPerProject int   // Max concurrent requests per project (0 = unlimited)
	MaxExecutionTimeMs       int   // Max request duration in ms; used for server write timeout (0 = use server default)
	MaxScanDocs              int   // Cap on list/query result limit per request (0 = no cap beyond existing 1000)
	MaxDatabaseSizeBytes     int64 // Max on-disk size per project in bytes (0 = unlimited)
}

// DirSize returns the total size in bytes of all files under dir (recursive).
func DirSize(dir string) (int64, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// ConcurrencyLimiter enforces max concurrent requests per project.
type ConcurrencyLimiter struct {
	limit  int
	counts sync.Map // projectID (string) -> *int32
}

// NewConcurrencyLimiter creates a limiter. limit 0 means unlimited (TryAcquire always succeeds).
func NewConcurrencyLimiter(limit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{limit: limit}
}

// TryAcquire increments the count for projectID. It returns false if at or over limit (caller should return 429).
// On success, caller must call Release(projectID) when done (e.g. defer).
func (c *ConcurrencyLimiter) TryAcquire(projectID string) bool {
	if c.limit <= 0 {
		return true
	}
	val, _ := c.counts.LoadOrStore(projectID, ptr32(0))
	counter := val.(*int32)
	n := atomic.AddInt32(counter, 1)
	if n > int32(c.limit) {
		atomic.AddInt32(counter, -1)
		return false
	}
	return true
}

// Release decrements the count for projectID.
func (c *ConcurrencyLimiter) Release(projectID string) {
	if c.limit <= 0 {
		return
	}
	val, ok := c.counts.Load(projectID)
	if !ok {
		return
	}
	atomic.AddInt32(val.(*int32), -1)
}

func ptr32(n int32) *int32 {
	return &n
}
