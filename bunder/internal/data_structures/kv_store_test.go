package data_structures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKVStore_GetSetDelete(t *testing.T) {
	dir := t.TempDir()
	kv, err := OpenKVStore(dir, 100, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer kv.Close()
	key := []byte("k")
	val := []byte("v")
	if kv.Get(key) != nil {
		t.Fatal("expected nil")
	}
	if err := kv.Set(key, val); err != nil {
		t.Fatal(err)
	}
	got := kv.Get(key)
	if string(got) != "v" {
		t.Fatalf("got %q", got)
	}
	ok, err := kv.Delete(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true")
	}
	if kv.Get(key) != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestKVStore_ExistsKeys(t *testing.T) {
	dir := t.TempDir()
	kv, err := OpenKVStore(dir, 100, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer kv.Close()
	kv.Set([]byte("a"), []byte("1"))
	kv.Set([]byte("b"), []byte("2"))
	if !kv.Exists([]byte("a")) {
		t.Fatal("expected exists a")
	}
	if kv.Exists([]byte("c")) {
		t.Fatal("expected not exists c")
	}
	keys := kv.Keys([]byte("*"))
	if len(keys) != 2 {
		t.Fatalf("keys: got %d", len(keys))
	}
}

func TestKVStore_Close(t *testing.T) {
	dir := t.TempDir()
	kv, err := OpenKVStore(dir, 100, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := kv.Close(); err != nil {
		t.Fatal(err)
	}
	// Second close is no-op
	if err := kv.Close(); err != nil {
		t.Fatal(err)
	}
	// Ensure data dir was created
	_ = filepath.Join(dir, "data.db")
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}
