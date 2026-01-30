package wal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kartikbazzad/docdb/internal/logger"
)

const (
	RotationSuffixActive = ""
	rotationSuffixPrefix = "."
)

// Rotator manages WAL segment rotation.
//
// It provides:
//   - Automatic rotation at size threshold
//   - Crash-safe rotation (atomic rename)
//   - Segment discovery and enumeration
//   - Multi-segment recovery support
type Rotator struct {
	basePath string // Base path without suffix
	maxSize  uint64 // Rotation threshold (0 = no rotation)
	fsync    bool   // Fsync on close
	logger   *logger.Logger
}

// NewRotator creates a new WAL rotator.
//
// Parameters:
//   - basePath: Base WAL path (e.g., "/data/wal/dbname.wal")
//   - maxSize: Maximum size before rotation (bytes, 0 = no limit)
//   - fsync: If true, fsync on close
//   - log: Logger instance
func NewRotator(basePath string, maxSize uint64, fsync bool, log *logger.Logger) *Rotator {
	return &Rotator{
		basePath: basePath,
		maxSize:  maxSize,
		fsync:    fsync,
		logger:   log,
	}
}

// ShouldRotate checks if rotation is needed based on current size.
func (r *Rotator) ShouldRotate(currentSize uint64) bool {
	if r.maxSize == 0 {
		return false
	}
	return currentSize >= r.maxSize
}

// Rotate performs WAL rotation.
//
// Process:
//  1. Sync and close current WAL
//  2. Determine next segment number
//  3. Rename current WAL with segment suffix
//  4. Return path to new WAL
//
// This is crash-safe: if rotation is interrupted, the old WAL
// remains usable with the original name.
func (r *Rotator) Rotate() (string, error) {
	segments, err := r.ListSegments()
	if err != nil {
		return "", fmt.Errorf("failed to list segments: %w", err)
	}

	nextSeq := 1
	if len(segments) > 0 {
		lastSeq, _ := r.extractSequenceNumber(filepath.Base(r.basePath), segments[len(segments)-1])
		nextSeq = lastSeq + 1
	}

	oldPath := r.basePath

	// Name segments as: <base>.wal.<n>, e.g. "testdb.wal.1"
	newPath := r.basePath + rotationSuffixPrefix + strconv.Itoa(nextSeq)

	r.logger.Info("Rotating WAL: %s -> %s", oldPath, newPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("active WAL not found: %s", oldPath)
		}
		return "", fmt.Errorf("failed to rename WAL: %w", err)
	}

	return oldPath, nil
}

// ListSegments returns all WAL segments sorted by sequence number.
//
// Returns paths in ascending order: dbname.wal.1, dbname.wal.2, ...
func (r *Rotator) ListSegments() ([]string, error) {
	dir := filepath.Dir(r.basePath)
	baseName := filepath.Base(r.basePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL directory: %w", err)
	}

	var segments []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !r.isSegmentName(baseName, name) {
			r.logger.Debug("Ignoring non-segment file: %s", name)
			continue
		}

		segments = append(segments, filepath.Join(dir, name))
		r.logger.Debug("Found segment: %s", name)
	}

	sort.Slice(segments, func(i, j int) bool {
		seqI, _ := r.extractSequenceNumber(baseName, segments[i])
		seqJ, _ := r.extractSequenceNumber(baseName, segments[j])
		return seqI < seqJ
	})

	// Log at Debug to avoid flooding console when listing is frequent (e.g. WAL size checks).
	if len(segments) > 0 {
		r.logger.Info("Found %d WAL segments", len(segments))
	} else {
		r.logger.Debug("Found %d WAL segments", len(segments))
	}
	return segments, nil
}

// GetAllWALPaths returns all WAL paths in recovery order.
//
// Order: oldest segment first, active WAL last.
// Example: [dbname.wal.1, dbname.wal.2, dbname.wal]
func (r *Rotator) GetAllWALPaths() ([]string, error) {
	segments, err := r.ListSegments()
	if err != nil {
		return nil, err
	}

	allPaths := make([]string, 0, len(segments)+1)
	allPaths = append(allPaths, segments...)

	activePath := r.basePath
	if _, err := os.Stat(activePath); err == nil {
		allPaths = append(allPaths, activePath)
	}

	return allPaths, nil
}

// CleanupOldSegments removes segments older than specified sequence number.
//
// This is for future use (v1.1 trimming).
func (r *Rotator) CleanupOldSegments(minSeq int) error {
	segments, err := r.ListSegments()
	if err != nil {
		return err
	}

	for _, segment := range segments {
		seq, err := r.extractSequenceNumber(filepath.Base(r.basePath), segment)
		if err != nil {
			continue
		}

		if seq >= minSeq {
			continue
		}

		r.logger.Info("Deleting old WAL segment: %s", segment)
		if err := os.Remove(segment); err != nil {
			r.logger.Warn("Failed to delete old WAL segment %s: %v", segment, err)
		}
	}

	return nil
}

// isSegmentName checks if a filename is a WAL segment.
func (r *Rotator) isSegmentName(baseName, filename string) bool {
	if len(filename) <= len(baseName) {
		return false
	}

	if filename[:len(baseName)] != baseName {
		return false
	}

	// Expect at least ".<number>" after baseName, but allow additional
	// dotted components (e.g. "testdb.wal.1.5") as long as the final
	// component is a pure numeric sequence.
	suffix := filename[len(baseName):]
	if len(suffix) < len(rotationSuffixPrefix)+1 {
		return false
	}
	if suffix[0:len(rotationSuffixPrefix)] != rotationSuffixPrefix {
		return false
	}

	// Take the part after the first dot and then the numeric tail after
	// the last dot.
	rest := suffix[len(rotationSuffixPrefix):]
	lastDot := strings.LastIndex(rest, rotationSuffixPrefix)
	numStr := rest
	if lastDot >= 0 {
		numStr = rest[lastDot+len(rotationSuffixPrefix):]
	}
	if numStr == "" {
		return false
	}
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// extractSequenceNumber extracts the sequence number from a segment path.
func (r *Rotator) extractSequenceNumber(baseName, segmentPath string) (int, error) {
	filename := filepath.Base(segmentPath)
	if len(filename) <= len(baseName)+len(rotationSuffixPrefix) {
		return 0, errors.New("invalid segment name")
	}

	if filename[:len(baseName)] != baseName {
		return 0, errors.New("invalid segment base name")
	}

	suffix := filename[len(baseName)+len(rotationSuffixPrefix):]
	if suffix == "" {
		return 0, errors.New("invalid segment suffix")
	}

	// If there are additional dots, only the last component is the sequence.
	lastDot := strings.LastIndex(suffix, rotationSuffixPrefix)
	seqStr := suffix
	if lastDot >= 0 {
		seqStr = suffix[lastDot+len(rotationSuffixPrefix):]
	}
	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		return 0, fmt.Errorf("invalid sequence number: %w", err)
	}

	return seq, nil
}
