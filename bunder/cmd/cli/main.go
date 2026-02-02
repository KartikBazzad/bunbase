// Bunder CLI is an interactive REPL for the Bunder server.
// It connects to a Bunder TCP address, reads lines from stdin, sends them as RESP arrays
// (e.g. *2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n), and prints the decoded response.
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	addr := "127.0.0.1:6379"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("Bunder CLI connected to %s. Type commands (GET key, SET key value, QUIT).\n", addr)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("bunder> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" || strings.ToUpper(line) == "QUIT" {
			break
		}
		// Send as RESP array of bulk strings
		parts := splitArgs(line)
		if len(parts) == 0 {
			continue
		}
		// Build RESP: *N\r\n $len\r\n arg\r\n ...
		buf := buildRESPArray(parts)
		if _, err := conn.Write(buf); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			break
		}
		// Read response
		br := bufio.NewReader(conn)
		resp, err := readRESPResponse(br)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read: %v\n", err)
			continue
		}
		fmt.Println(resp)
	}
}

func splitArgs(line string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '"' || c == '\'':
			inQuote = !inQuote
		case (c == ' ' || c == '\t') && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func buildRESPArray(args []string) []byte {
	var b []byte
	b = append(b, '*')
	b = append(b, fmt.Sprintf("%d", len(args))...)
	b = append(b, '\r', '\n')
	for _, a := range args {
		b = append(b, '$')
		b = append(b, fmt.Sprintf("%d", len(a))...)
		b = append(b, '\r', '\n')
		b = append(b, a...)
		b = append(b, '\r', '\n')
	}
	return b
}

func readRESPResponse(r *bufio.Reader) (string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	switch b {
	case '+':
		line, err := readLine(r)
		return string(line), err
	case '-':
		line, err := readLine(r)
		return "ERR " + string(line), err
	case ':':
		line, err := readLine(r)
		return string(line), err
	case '$':
		line, err := readLine(r)
		if err != nil {
			return "", err
		}
		n, _ := strconv.Atoi(string(line))
		if n == -1 {
			return "(nil)", nil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return "", err
		}
		count, _ := strconv.Atoi(string(line))
		var parts []string
		for i := 0; i < count; i++ {
			bb, _ := r.ReadByte()
			if bb == '$' {
				ln, _ := readLine(r)
				nn, _ := strconv.Atoi(string(ln))
				buf := make([]byte, nn+2)
				r.Read(buf)
				parts = append(parts, string(buf[:nn]))
			}
		}
		return strings.Join(parts, " "), nil
	default:
		return "", fmt.Errorf("unknown type %c", b)
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	var out []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '\n' {
			if len(out) > 0 && out[len(out)-1] == '\r' {
				out = out[:len(out)-1]
			}
			return out, nil
		}
		out = append(out, b)
	}
}
