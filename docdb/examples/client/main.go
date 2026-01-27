package main

import (
	"encoding/hex"
	"encoding/json"
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
		fmt.Printf("  Document %d data (hex): %s\n", docID, hex.EncodeToString(data))
		fmt.Printf("  Document %d data (length): %d bytes\n", docID, len(data))
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
		fmt.Printf("  Document 3 data (hex): %s\n", hex.EncodeToString(data))
		fmt.Printf("  Document 3 data (length): %d bytes\n", len(data))
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
		fmt.Printf("  Document %d data (hex): %s\n", docID, hex.EncodeToString(data))
		fmt.Printf("  Document %d data (length): %d bytes\n", docID, len(data))
	}
	fmt.Println()

	fmt.Println("=== JSON Document Operations ===")
	fmt.Println()

	// Create JSON documents
	type User struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Age      int    `json:"age"`
		Active   bool   `json:"active"`
		Tags     []string `json:"tags"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	jsonDocs := []User{
		{ID: 100, Name: "Alice Johnson", Email: "alice@example.com", Age: 28, Active: true, Tags: []string{"developer", "go"}, Metadata: map[string]interface{}{"role": "backend", "level": "senior"}},
		{ID: 101, Name: "Bob Smith", Email: "bob@example.com", Age: 32, Active: true, Tags: []string{"designer", "ui"}, Metadata: map[string]interface{}{"role": "frontend", "level": "mid"}},
		{ID: 102, Name: "Charlie Brown", Email: "charlie@example.com", Age: 25, Active: false, Tags: []string{"intern"}, Metadata: map[string]interface{}{"role": "fullstack", "level": "junior"}},
	}

	fmt.Println("Creating JSON documents...")
	for i, user := range jsonDocs {
		docID := uint64(100 + i) // Use IDs 100, 101, 102
		
		jsonData, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			fmt.Printf("Failed to marshal JSON for document %d: %v\n", docID, err)
			continue
		}

		err = cli.Create(dbID, docID, jsonData)
		if err != nil {
			fmt.Printf("Failed to create JSON document %d: %v\n", docID, err)
			continue
		}

		fmt.Printf("  Created JSON document %d:\n", docID)
		fmt.Printf("    %s\n", string(jsonData))
	}
	fmt.Println()

	fmt.Println("Reading JSON documents...")
	for i := 0; i < len(jsonDocs); i++ {
		docID := uint64(100 + i)

		data, err := cli.Read(dbID, docID)
		if err != nil {
			fmt.Printf("Failed to read JSON document %d: %v\n", docID, err)
			continue
		}

		fmt.Printf("  Document %d (raw): %s\n", docID, string(data))
		fmt.Printf("  Document %d (hex): %s\n", docID, hex.EncodeToString(data))
		fmt.Printf("  Document %d (length): %d bytes\n", docID, len(data))

		// Parse and display as structured JSON
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			fmt.Printf("  Failed to parse JSON: %v\n", err)
		} else {
			fmt.Printf("  Document %d (parsed):\n", docID)
			fmt.Printf("    ID: %d\n", user.ID)
			fmt.Printf("    Name: %s\n", user.Name)
			fmt.Printf("    Email: %s\n", user.Email)
			fmt.Printf("    Age: %d\n", user.Age)
			fmt.Printf("    Active: %v\n", user.Active)
			fmt.Printf("    Tags: %v\n", user.Tags)
			fmt.Printf("    Metadata: %v\n", user.Metadata)
		}
		fmt.Println()
	}

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
