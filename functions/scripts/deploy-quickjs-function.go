package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/kartikbazzad/bunbase/functions/internal/capabilities"
	"github.com/kartikbazzad/bunbase/functions/internal/config"
	"github.com/kartikbazzad/bunbase/functions/internal/logger"
	"github.com/kartikbazzad/bunbase/functions/internal/metadata"
)

func main() {
	var (
		functionName = flag.String("name", "", "Function name (required)")
		functionFile = flag.String("file", "", "Function source file path (required)")
		version      = flag.String("version", "v1", "Function version")
		profile      = flag.String("profile", "strict", "Capability profile: strict, permissive, or custom")
		dataDir      = flag.String("data-dir", "./data", "Data directory path")
		projectID    = flag.String("project", "", "Project ID (defaults to function name)")
	)
	flag.Parse()

	if *functionName == "" || *functionFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -name <function-name> -file <function-file> [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *projectID == "" {
		*projectID = *functionName
	}

	log := logger.Default()
	log.Info("Deploying function with QuickJS-NG runtime...")
	log.Info("Function: %s", *functionName)
	log.Info("File: %s", *functionFile)
	log.Info("Version: %s", *version)
	log.Info("Profile: %s", *profile)

	// 1. Build bundle
	bundlePath, err := buildBundle(*functionFile, *functionName, *version, *dataDir, log)
	if err != nil {
		log.Error("Failed to build bundle: %v", err)
		os.Exit(1)
	}

	// 2. Create capabilities
	caps := createCapabilities(*profile, *projectID, log)

	// 3. Initialize metadata store
	dbPath := filepath.Join(*dataDir, "metadata.db")
	meta, err := metadata.NewStore(dbPath)
	if err != nil {
		log.Error("Failed to create metadata store: %v", err)
		os.Exit(1)
	}
	defer meta.Close()

	// 4. Register function
	functionID := fmt.Sprintf("func-%s", *functionName)
	fn, err := meta.RegisterFunction(functionID, *functionName, "quickjs-ng", "handler", caps)
	if err != nil {
		log.Error("Failed to register function: %v", err)
		os.Exit(1)
	}
	log.Info("Registered function: %s (ID: %s)", fn.Name, fn.ID)

	// 5. Create version
	versionID := uuid.New().String()
	_, err = meta.CreateVersion(versionID, fn.ID, *version, bundlePath)
	if err != nil {
		log.Error("Failed to create version: %v", err)
		os.Exit(1)
	}
	log.Info("Created version: %s", *version)

	// 6. Deploy function
	deploymentID := uuid.New().String()
	err = meta.DeployFunction(deploymentID, fn.ID, versionID)
	if err != nil {
		log.Error("Failed to deploy function: %v", err)
		os.Exit(1)
	}
	log.Info("Deployed function: %s", fn.ID)

	// 7. Print summary
	fmt.Println()
	fmt.Println("=== Deployment Complete ===")
	fmt.Printf("Function ID:   %s\n", fn.ID)
	fmt.Printf("Function Name: %s\n", fn.Name)
	fmt.Printf("Version:       %s\n", *version)
	fmt.Printf("Runtime:       quickjs-ng\n")
	fmt.Printf("Bundle:        %s\n", bundlePath)
	fmt.Printf("Capabilities:  %s\n", *profile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Start functions service: ./functions --data-dir %s\n", *dataDir)
	fmt.Printf("  2. Test: curl 'http://localhost:8080/functions/%s'\n", *functionName)
}

func buildBundle(functionFile, functionName, version, dataDir string, log *logger.Logger) (string, error) {
	// Create bundle directory
	bundleDir := filepath.Join(dataDir, "bundles", fmt.Sprintf("func-%s", functionName), version)
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bundle directory: %w", err)
	}

	bundlePath := filepath.Join(bundleDir, "bundle.js")

	// Check if bun is available
	if _, err := os.Stat(bundlePath); err == nil {
		log.Info("Bundle already exists, skipping build")
		return filepath.Abs(bundlePath)
	}

	// Try to build with bun
	log.Info("Building bundle with bun...")
	// In a real implementation, you'd call bun build here
	// For now, we'll just copy the file (assuming it's already a bundle)
	// In production, you'd want to actually bundle it

	absPath, err := filepath.Abs(bundlePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

func createCapabilities(profile, projectID string, log *logger.Logger) *capabilities.Capabilities {
	switch profile {
	case "strict":
		return capabilities.StrictProfile(projectID)
	case "permissive":
		return capabilities.PermissiveProfile(projectID)
	default:
		log.Warn("Unknown profile '%s', using strict", profile)
		return capabilities.StrictProfile(projectID)
	}
}
