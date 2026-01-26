package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kartikbazzad/docdb/pkg/client"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: example <socket-path>")
		os.Exit(1)
	}

	socketPath := os.Args[1]
	cli := client.New(socketPath)
	defer cli.Close()

	fmt.Println("=== DocDB Example Client ===")
	fmt.Println()

	dbName := fmt.Sprintf("example_db_%d", time.Now().Unix())

	fmt.Printf("Creating database: %s\n", dbName)
	dbID, err := cli.OpenDB(dbName)
	if err != nil {
		fmt.Printf("Failed to create database: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created database with ID: %d\n", dbID)
	fmt.Println()

	fmt.Println("Creating documents...")
	for i := 1; i <= 5; i++ {
		payload := []byte(fmt.Sprintf("Document %d payload", i))
		docID := uint64(i)

		err := cli.Create(dbID, docID, payload)
		if err != nil {
			fmt.Printf("Failed to create document %d: %v\n", docID, err)
			continue
		}

		fmt.Printf("  Created document %d: %s\n", docID, string(payload))
	}
	fmt.Println()

	fmt.Println("Reading documents...")
	for i := 1; i <= 5; i++ {
		docID := uint64(i)

		data, err := cli.Read(dbID, docID)
		if err != nil {
			fmt.Printf("Failed to read document %d: %v\n", docID, err)
			continue
		}

		fmt.Printf("  Document %d: %s\n", docID, string(data))
	}
	fmt.Println()

	fmt.Println("Updating document 3...")
	newPayload := []byte("Updated payload for document 3")
	err = cli.Update(dbID, 3, newPayload)
	if err != nil {
		fmt.Printf("Failed to update document 3: %v\n", err)
	} else {
		fmt.Printf("  Document 3 updated: %s\n", string(newPayload))
	}
	fmt.Println()

	fmt.Println("Reading updated document 3...")
	data, err := cli.Read(dbID, 3)
	if err != nil {
		fmt.Printf("Failed to read document 3: %v\n", err)
	} else {
		fmt.Printf("  Document 3: %s\n", string(data))
	}
	fmt.Println()

	fmt.Println("Deleting document 5...")
	err = cli.Delete(dbID, 5)
	if err != nil {
		fmt.Printf("Failed to delete document 5: %v\n", err)
	} else {
		fmt.Println("  Document 5 deleted")
	}
	fmt.Println()

	fmt.Println("Reading documents after delete...")
	for i := 1; i <= 5; i++ {
		docID := uint64(i)

		data, err := cli.Read(dbID, docID)
		if err != nil {
			fmt.Printf("  Document %d: not found (as expected)\n", docID)
			continue
		}

		fmt.Printf("  Document %d: %s\n", docID, string(data))
	}
	fmt.Println()

	fmt.Println("Getting pool stats...")
	stats, err := cli.Stats()
	if err != nil {
		fmt.Printf("Failed to get stats: %v\n", err)
	} else {
		fmt.Printf("  Total DBs: %d\n", stats.TotalDBs)
		fmt.Printf("  Active DBs: %d\n", stats.ActiveDBs)
		fmt.Printf("  Memory Used: %d bytes\n", stats.MemoryUsed)
		fmt.Printf("  Memory Capacity: %d bytes\n", stats.MemoryCapacity)
	}
	fmt.Println()

	fmt.Println("Closing database...")
	err = cli.CloseDB(dbID)
	if err != nil {
		fmt.Printf("Failed to close database: %v\n", err)
	} else {
		fmt.Println("  Database closed")
	}
	fmt.Println()

	fmt.Println("=== Example Complete ===")
}
