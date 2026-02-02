// Package server implements the Bunder TCP server (RESP protocol), HTTP API, and command handler.
package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// RESP (Redis Serialization Protocol) type prefixes.
const (
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
)

var crlf = []byte("\r\n")

// ReadRESP reads one RESP value from r (array of bulk strings for client commands).
func ReadRESP(r *bufio.Reader) (interface{}, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case TypeArray:
		return readArray(r)
	case TypeBulkString:
		return readBulkString(r)
	case TypeSimpleString:
		return readLine(r)
	case TypeError:
		line, _ := readLine(r)
		return nil, fmt.Errorf("resp error: %s", line)
	case TypeInteger:
		line, _ := readLine(r)
		n, _ := strconv.ParseInt(string(line), 10, 64)
		return n, nil
	default:
		return nil, fmt.Errorf("resp: unknown type %c", b)
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		return line[:len(line)-2], nil
	}
	return line[:len(line)-1], nil
}

func readArray(r *bufio.Reader) ([]interface{}, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(string(line))
	if err != nil || n < 0 {
		return nil, fmt.Errorf("resp: invalid array length")
	}
	out := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != TypeBulkString {
			return nil, fmt.Errorf("resp: expected bulk string in array")
		}
		val, err := readBulkString(r)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	return out, nil
}

func readBulkString(r *bufio.Reader) ([]byte, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	lenNum, err := strconv.Atoi(string(line))
	if err != nil || lenNum < -1 {
		return nil, fmt.Errorf("resp: invalid bulk string length")
	}
	if lenNum == -1 {
		return nil, nil // nil bulk string
	}
	buf := make([]byte, lenNum+2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if buf[lenNum] != '\r' || buf[lenNum+1] != '\n' {
		return nil, fmt.Errorf("resp: bulk string not CRLF terminated")
	}
	return buf[:lenNum], nil
}

// ParseCommand parses a RESP array into command name and args (all []byte).
func ParseCommand(arr []interface{}) (cmd string, args [][]byte, err error) {
	if len(arr) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}
	cmdBytes, ok := arr[0].([]byte)
	if !ok {
		return "", nil, fmt.Errorf("command not bulk string")
	}
	cmd = string(bytes.ToUpper(cmdBytes))
	args = make([][]byte, 0, len(arr)-1)
	for i := 1; i < len(arr); i++ {
		a, ok := arr[i].([]byte)
		if !ok {
			return "", nil, fmt.Errorf("arg %d not bulk string", i)
		}
		args = append(args, a)
	}
	return cmd, args, nil
}

// WriteRESP writes a RESP value to w.
func WriteRESP(w io.Writer, v interface{}) error {
	switch x := v.(type) {
	case nil:
		return writeBulkString(w, nil)
	case []byte:
		return writeBulkString(w, x)
	case string:
		return writeSimpleString(w, x)
	case int:
		return writeInteger(w, int64(x))
	case int64:
		return writeInteger(w, x)
	case error:
		return writeError(w, x.Error())
	case []interface{}:
		return writeArray(w, x)
	case [][]byte:
		arr := make([]interface{}, len(x))
		for i, b := range x {
			arr[i] = b
		}
		return writeArray(w, arr)
	default:
		return fmt.Errorf("resp write: unsupported type %T", v)
	}
}

func writeSimpleString(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "+%s\r\n", s)
	return err
}

func writeError(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "-%s\r\n", s)
	return err
}

func writeInteger(w io.Writer, n int64) error {
	_, err := fmt.Fprintf(w, ":%d\r\n", n)
	return err
}

func writeBulkString(w io.Writer, b []byte) error {
	if b == nil {
		_, err := w.Write([]byte("$-1\r\n"))
		return err
	}
	if _, err := fmt.Fprintf(w, "$%d\r\n", len(b)); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	_, err := w.Write(crlf)
	return err
}

func writeArray(w io.Writer, arr []interface{}) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(arr)); err != nil {
		return err
	}
	for _, v := range arr {
		if err := WriteRESP(w, v); err != nil {
			return err
		}
	}
	return nil
}
