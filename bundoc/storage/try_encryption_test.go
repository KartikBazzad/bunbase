package storage

import (
	"bytes"
	"os"
	"testing"
)

func TestEncryptionAtRest(t *testing.T) {
	// 1. Setup
	key := []byte("12345678901234567890123456789012") // 32 bytes
	tmpEnc := "test_enc.db"
	tmpPlain := "test_plain.db"
	defer os.Remove(tmpEnc)
	defer os.Remove(tmpPlain)

	// 2. Write Encrypted Page
	pagerEnc, err := NewPager(tmpEnc, key)
	if err != nil {
		t.Fatalf("Failed to create encrypted pager: %v", err)
	}

	pageID, _ := pagerEnc.AllocatePage()
	page := NewPage(pageID, PageTypeLeaf)
	data := []byte("Sensitive Data Secret")
	copy(page.Data[:], data)

	if err := pagerEnc.WritePage(page); err != nil {
		t.Fatalf("Failed to write encrypted page: %v", err)
	}
	pagerEnc.Close()

	// 3. Write Plaintext Page (Control)
	pagerPlain, err := NewPager(tmpPlain, nil)
	if err != nil {
		t.Fatalf("Failed to create plain pager: %v", err)
	}

	pageID2, _ := pagerPlain.AllocatePage()
	page2 := NewPage(pageID2, PageTypeLeaf)
	copy(page2.Data[:], data)

	if err := pagerPlain.WritePage(page2); err != nil {
		t.Fatalf("Failed to write plain page: %v", err)
	}
	pagerPlain.Close()

	// 4. Compare Disk Content
	rawEnc, _ := os.ReadFile(tmpEnc)
	rawPlain, _ := os.ReadFile(tmpPlain)

	// The encrypted file should be slightly larger (overhead) and NOT contain the plaintext string
	if len(rawEnc) <= len(rawPlain) {
		t.Errorf("Encrypted file size %d should be > plain file size %d", len(rawEnc), len(rawPlain))
	}

	if bytes.Contains(rawEnc, data) {
		t.Error("Encrypted file contains plaintext data! Encryption failed.")
	}
	if !bytes.Contains(rawPlain, data) {
		t.Error("Plain file missing data? Something wrong with test.")
	}

	// 5. Verify Read Back (Decrypt)
	pagerRead, err := NewPager(tmpEnc, key)
	if err != nil {
		t.Fatalf("Failed to open pager for read: %v", err)
	}

	readPage, err := pagerRead.ReadPage(pageID)
	if err != nil {
		t.Fatalf("Failed to read encrypted page: %v", err)
	}

	if !bytes.Contains(readPage.Data[:], data) {
		t.Error("Failed to decrypt data correctly.")
	}
	pagerRead.Close()
}
