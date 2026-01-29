package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kartikbazzad/docdb/tests/load"
)

func main() {
	resultsDir := flag.String("results-dir", "./matrix_results", "Matrix results base directory (containing json/, csv_*/, reports/)")
	outputPath := flag.String("output", "./matrix_results/reports/analysis.md", "Output path for analysis report")
	flag.Parse()

	if *resultsDir == "" {
		log.Fatalf("Must specify -results-dir")
	}

	// Analyze matrix results
	analysis, err := load.AnalyzeMatrix(*resultsDir)
	if err != nil {
		log.Fatalf("Failed to analyze matrix: %v", err)
	}

	// Ensure reports directory exists
	reportDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		log.Fatalf("Failed to create reports directory: %v", err)
	}

	// Generate report
	if err := analysis.GenerateReport(*outputPath); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	fmt.Printf("Analysis complete. Report written to %s\n", *outputPath)
}
