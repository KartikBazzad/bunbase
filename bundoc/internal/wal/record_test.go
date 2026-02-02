package wal

import (
	"bytes"
	"testing"
	"time"
)

func TestRecordEncodeDecode(t *testing.T) {
	// Create a test record
	original := &Record{
		LSN:       LSN(12345),
		TxnID:     67890,
		Type:      RecordTypeInsert,
		Key:       []byte("test_key"),
		Value:     []byte("test_value"),
		PrevLSN:   LSN(12340),
		Timestamp: time.Now().UnixNano(),
	}

	// Encode
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Failed to encode record: %v", err)
	}

	// Verify size
	expectedSize := original.Size()
	if len(encoded) != expectedSize {
		t.Errorf("Encoded size mismatch: expected %d, got %d", expectedSize, len(encoded))
	}

	// Decode
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Failed to decode record: %v", err)
	}

	// Verify fields
	if decoded.LSN != original.LSN {
		t.Errorf("LSN mismatch: expected %d, got %d", original.LSN, decoded.LSN)
	}
	if decoded.TxnID != original.TxnID {
		t.Errorf("TxnID mismatch: expected %d, got %d", original.TxnID, decoded.TxnID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: expected %d, got %d", original.Type, decoded.Type)
	}
	if !bytes.Equal(decoded.Key, original.Key) {
		t.Errorf("Key mismatch: expected %s, got %s", original.Key, decoded.Key)
	}
	if !bytes.Equal(decoded.Value, original.Value) {
		t.Errorf("Value mismatch: expected %s, got %s", original.Value, decoded.Value)
	}
	if decoded.PrevLSN != original.PrevLSN {
		t.Errorf("PrevLSN mismatch: expected %d, got %d", original.PrevLSN, decoded.PrevLSN)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", original.Timestamp, decoded.Timestamp)
	}
}

func TestRecordTypes(t *testing.T) {
	types := []RecordType{
		RecordTypeInsert,
		RecordTypeUpdate,
		RecordTypeDelete,
		RecordTypeCommit,
		RecordTypeAbort,
		RecordTypeCheckpoint,
	}

	for _, recordType := range types {
		record := &Record{
			LSN:       LSN(1),
			TxnID:     1,
			Type:      recordType,
			Key:       []byte("key"),
			Value:     []byte("value"),
			PrevLSN:   LSN(0),
			Timestamp: time.Now().UnixNano(),
		}

		encoded, err := record.Encode()
		if err != nil {
			t.Fatalf("Failed to encode record type %d: %v", recordType, err)
		}

		decoded, err := Decode(encoded)
		if err != nil {
			t.Fatalf("Failed to decode record type %d: %v", recordType, err)
		}

		if decoded.Type != recordType {
			t.Errorf("Record type mismatch: expected %d, got %d", recordType, decoded.Type)
		}
	}
}

func TestRecordCRCValidation(t *testing.T) {
	record := &Record{
		LSN:       LSN(100),
		TxnID:     200,
		Type:      RecordTypeInsert,
		Key:       []byte("key"),
		Value:     []byte("value"),
		PrevLSN:   LSN(99),
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := record.Encode()
	if err != nil {
		t.Fatalf("Failed to encode record: %v", err)
	}

	// Corrupt the data (change a byte in the middle)
	encoded[RecordHeaderSize/2]++

	// Should fail CRC check
	_, err = Decode(encoded)
	if err == nil {
		t.Error("Expected CRC error for corrupted data, got nil")
	}
}

func TestRecordEmptyKeyValue(t *testing.T) {
	record := &Record{
		LSN:       LSN(1),
		TxnID:     1,
		Type:      RecordTypeCommit,
		Key:       []byte{},
		Value:     []byte{},
		PrevLSN:   LSN(0),
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := record.Encode()
	if err != nil {
		t.Fatalf("Failed to encode record with empty key/value: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Failed to decode record with empty key/value: %v", err)
	}

	if len(decoded.Key) != 0 || len(decoded.Value) != 0 {
		t.Error("Expected empty key and value")
	}
}

func TestRecordLargeData(t *testing.T) {
	// Create large key and value (1KB each)
	largeKey := bytes.Repeat([]byte("k"), 1024)
	largeValue := bytes.Repeat([]byte("v"), 1024)

	record := &Record{
		LSN:       LSN(999),
		TxnID:     888,
		Type:      RecordTypeUpdate,
		Key:       largeKey,
		Value:     largeValue,
		PrevLSN:   LSN(998),
		Timestamp: time.Now().UnixNano(),
	}

	encoded, err := record.Encode()
	if err != nil {
		t.Fatalf("Failed to encode large record: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Failed to decode large record: %v", err)
	}

	if !bytes.Equal(decoded.Key, largeKey) {
		t.Error("Large key mismatch")
	}
	if !bytes.Equal(decoded.Value, largeValue) {
		t.Error("Large value mismatch")
	}
}
