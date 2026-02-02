package concurrency

import (
	"testing"
)

func TestShardedMap_GetSetDelete(t *testing.T) {
	m := NewShardedMap(8)
	key := []byte("foo")
	val := []byte("bar")
	if m.Get(key) != nil {
		t.Fatal("expected nil for missing key")
	}
	m.Set(key, val)
	got := m.Get(key)
	if string(got) != "bar" {
		t.Fatalf("got %q", got)
	}
	if !m.Delete(key) {
		t.Fatal("expected true from Delete")
	}
	if m.Get(key) != nil {
		t.Fatal("expected nil after delete")
	}
	if m.Delete(key) {
		t.Fatal("expected false on second delete")
	}
}

func TestShardedMap_Exists(t *testing.T) {
	m := NewShardedMap(4)
	if m.Exists([]byte("x")) {
		t.Fatal("expected false")
	}
	m.Set([]byte("x"), []byte("1"))
	if !m.Exists([]byte("x")) {
		t.Fatal("expected true")
	}
}

func TestShardedMap_Count(t *testing.T) {
	m := NewShardedMap(4)
	if m.Count() != 0 {
		t.Fatalf("count 0: got %d", m.Count())
	}
	m.Set([]byte("a"), []byte("1"))
	m.Set([]byte("b"), []byte("2"))
	if m.Count() != 2 {
		t.Fatalf("count 2: got %d", m.Count())
	}
	m.Delete([]byte("a"))
	if m.Count() != 1 {
		t.Fatalf("count 1: got %d", m.Count())
	}
}
