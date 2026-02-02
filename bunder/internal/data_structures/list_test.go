package data_structures

import (
	"testing"
)

func TestList_LPushRPop(t *testing.T) {
	l := NewList()
	l.LPush([]byte("a"), []byte("b"))
	if l.LLen() != 2 {
		t.Fatalf("len: got %d", l.LLen())
	}
	v := l.LPop()
	if string(v) != "b" {
		t.Fatalf("LPop: got %q", v)
	}
	v = l.RPop()
	if string(v) != "a" {
		t.Fatalf("RPop: got %q", v)
	}
	if l.LPop() != nil {
		t.Fatal("expected nil")
	}
}

func TestList_LRange(t *testing.T) {
	l := NewList()
	l.RPush([]byte("x"), []byte("y"), []byte("z"))
	r := l.LRange(0, -1)
	if len(r) != 3 {
		t.Fatalf("range: got %d", len(r))
	}
	if string(r[0]) != "x" || string(r[1]) != "y" || string(r[2]) != "z" {
		t.Fatalf("range: got %q", r)
	}
}

func TestList_BytesRoundtrip(t *testing.T) {
	l := NewList()
	l.RPush([]byte("a"), []byte("b"))
	b := l.Bytes()
	l2 := NewList()
	if err := l2.LoadFromBytes(b); err != nil {
		t.Fatal(err)
	}
	if l2.LLen() != 2 {
		t.Fatalf("roundtrip len: got %d", l2.LLen())
	}
	if string(l2.LIndex(0)) != "a" || string(l2.LIndex(1)) != "b" {
		t.Fatalf("roundtrip: got %q %q", l2.LIndex(0), l2.LIndex(1))
	}
}
