package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/kartikbazzad/docdb/cmd/docdbsh/parser"
	"github.com/kartikbazzad/docdb/cmd/docdbsh/shell"
)

const (
	prompt = "> "
)

func main() {
	socketPath := flag.String("socket", "/tmp/docdb.sock", "Unix socket path")
	flag.Parse()

	fmt.Printf("DocDB Shell v0\n")
	fmt.Printf("Connecting to %s...\n", *socketPath)

	sh, err := shell.NewShell(*socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize shell: %v\n", err)
		os.Exit(1)
	}
	defer sh.Close()

	if err := sh.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connected. Type '.help' for commands.\n\n")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nInterrupted. Exiting...")
		sh.Close()
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(prompt)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		line = line[:len(line)-1]
		if line == "" {
			continue
		}

		cmd, err := parser.Parse(line)
		if err != nil {
			fmt.Fprintln(os.Stdout, "ERROR")
			fmt.Fprintln(os.Stdout, err.Error())
			fmt.Println()
			continue
		}

		result := sh.Execute(cmd)
		if result.IsExit() {
			return
		}
		result.Print(os.Stdout)
		fmt.Println()
	}
}
