package load

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureOutputDirs ensures the standard output directory layout exists under base:
//
//	<base>/json
//	<base>/csv_global
//	<base>/csv_dbs
//	<base>/reports
//
// It returns the full paths to these subdirectories.
func EnsureOutputDirs(base string) (jsonDir, csvGlobalDir, csvDbsDir, reportsDir string, err error) {
	if base == "" {
		base = "."
	}

	jsonDir = filepath.Join(base, "json")
	csvGlobalDir = filepath.Join(base, "csv_global")
	csvDbsDir = filepath.Join(base, "csv_dbs")
	reportsDir = filepath.Join(base, "reports")

	dirs := []string{jsonDir, csvGlobalDir, csvDbsDir, reportsDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", "", "", "", fmt.Errorf("failed to create output directory %s: %w", d, err)
		}
	}

	return jsonDir, csvGlobalDir, csvDbsDir, reportsDir, nil
}
