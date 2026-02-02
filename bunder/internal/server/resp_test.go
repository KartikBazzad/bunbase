package server

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReadRESP_Array(t *testing.T) {
	// *2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n
	input := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	r := bufio.NewReader(bytes.NewReader([]byte(input)))
	val, err := ReadRESP(r)
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := val.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", val)
	}
	if len(arr) != 2 {
		t.Fatalf("len: got %d", len(arr))
	}
	cmd, args, err := ParseCommand(arr)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "GET" {
		t.Fatalf("cmd: got %q", cmd)
	}
	if len(args) != 1 || string(args[0]) != "foo" {
		t.Fatalf("args: %q", args)
	}
}

func TestWriteRESP(t *testing.T) {
	var b bytes.Buffer
	if err := WriteRESP(&b, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if b.String() != "$5\r\nhello\r\n" {
		t.Fatalf("got %q", b.String())
	}
	b.Reset()
	if err := WriteRESP(&b, "OK"); err != nil {
		t.Fatal(err)
	}
	if b.String() != "+OK\r\n" {
		t.Fatalf("got %q", b.String())
	}
	b.Reset()
	if err := WriteRESP(&b, int64(42)); err != nil {
		t.Fatal(err)
	}
	if b.String() != ":42\r\n" {
		t.Fatalf("got %q", b.String())
	}
}
