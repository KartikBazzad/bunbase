package wal

import (
	"encoding/binary"
	"hash/crc32"

	"github.com/kartikbazzad/docdb/internal/types"
)

var byteOrder = binary.LittleEndian

const (
	DefaultCollection    = "_default"
	MaxCollectionNameLen = 64
)

// EncodeRecord encodes a WAL record in v0.1 format (backward compatibility).
// Collection is empty, will be treated as "_default" during decode.
func EncodeRecord(txID, dbID, docID uint64, opType types.OperationType, payload []byte) ([]byte, error) {
	return EncodeRecordV2(txID, dbID, "", docID, opType, payload)
}

// EncodeRecordV2 encodes a WAL record in v0.2 format with collection name.
func EncodeRecordV2(txID, dbID uint64, collection string, docID uint64, opType types.OperationType, payload []byte) ([]byte, error) {
	payloadLen := uint32(len(payload))
	if uint32(payloadLen) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	// Normalize empty collection to default
	if collection == "" {
		collection = DefaultCollection
	}

	collectionBytes := []byte(collection)
	collectionLen := uint16(len(collectionBytes))
	if collectionLen > MaxCollectionNameLen {
		return nil, ErrPayloadTooLarge
	}

	// Calculate total length: header + collection name + payload + CRC
	totalLen := RecordOverheadV2Min + uint64(collectionLen) + uint64(payloadLen)
	buf := make([]byte, totalLen)

	offset := 0

	byteOrder.PutUint64(buf[offset:], totalLen)
	offset += RecordLenSize

	byteOrder.PutUint64(buf[offset:], txID)
	offset += TxIDSize

	byteOrder.PutUint64(buf[offset:], dbID)
	offset += DBIDSize

	// v0.2: collection name
	byteOrder.PutUint16(buf[offset:], collectionLen)
	offset += CollectionLenSize
	if collectionLen > 0 {
		copy(buf[offset:], collectionBytes)
		offset += int(collectionLen)
	}

	buf[offset] = byte(opType)
	offset += OpTypeSize

	byteOrder.PutUint64(buf[offset:], docID)
	offset += DocIDSize

	byteOrder.PutUint32(buf[offset:], payloadLen)
	offset += PayloadLenSize

	if len(payload) > 0 {
		copy(buf[offset:], payload)
		offset += len(payload)
	}

	crc := crc32.ChecksumIEEE(buf[:offset])
	byteOrder.PutUint32(buf[offset:], crc)

	return buf, nil
}

func DecodeRecord(data []byte) (*types.WALRecord, error) {
	if len(data) < RecordOverheadV1 {
		return nil, ErrCorruptRecord
	}

	offset := 0
	recordLen := byteOrder.Uint64(data[offset:])
	offset += RecordLenSize

	if uint64(len(data)) != recordLen {
		return nil, ErrCorruptRecord
	}

	storedCRC := byteOrder.Uint32(data[len(data)-CRCSize:])
	computedCRC := crc32.ChecksumIEEE(data[:len(data)-CRCSize])

	if storedCRC != computedCRC {
		return nil, ErrCRCMismatch
	}

	txID := byteOrder.Uint64(data[offset:])
	offset += TxIDSize

	dbID := byteOrder.Uint64(data[offset:])
	offset += DBIDSize

	// Detect v0.1 vs v0.2 format
	// v0.1: After DBID comes OpType directly
	// v0.2: After DBID comes CollectionLen, then CollectionName, then OpType
	// We detect by checking if there's enough space for CollectionLen before OpType

	remainingBeforeOpType := recordLen - uint64(offset) - CRCSize
	minV2Size := CollectionLenSize + OpTypeSize + DocIDSize + PayloadLenSize

	var collection string
	if remainingBeforeOpType >= uint64(minV2Size) {
		// Try to read as v0.2 format
		if offset+CollectionLenSize <= len(data) {
			collectionLen := byteOrder.Uint16(data[offset:])
			offset += CollectionLenSize

			if collectionLen > 0 && offset+int(collectionLen) <= len(data) {
				collectionBytes := data[offset : offset+int(collectionLen)]
				collection = string(collectionBytes)
				offset += int(collectionLen)
			} else if collectionLen == 0 {
				collection = DefaultCollection
			}
		}
	}

	// If we didn't read collection (v0.1 format), default to "_default"
	if collection == "" {
		collection = DefaultCollection
	}

	if offset+OpTypeSize > len(data) {
		return nil, ErrCorruptRecord
	}

	opType := types.OperationType(data[offset])
	offset += OpTypeSize

	if offset+DocIDSize > len(data) {
		return nil, ErrCorruptRecord
	}

	docID := byteOrder.Uint64(data[offset:])
	offset += DocIDSize

	if offset+PayloadLenSize > len(data) {
		return nil, ErrCorruptRecord
	}

	payloadLen := byteOrder.Uint32(data[offset:])
	offset += PayloadLenSize

	var payload []byte
	if payloadLen > 0 {
		if offset+int(payloadLen) > len(data) {
			return nil, ErrCorruptRecord
		}
		payload = make([]byte, payloadLen)
		copy(payload, data[offset:offset+int(payloadLen)])
	}

	return &types.WALRecord{
		Length:     recordLen,
		TxID:       txID,
		DBID:       dbID,
		Collection: collection,
		OpType:     opType,
		DocID:      docID,
		PayloadLen: payloadLen,
		Payload:    payload,
		CRC:        storedCRC,
	}, nil
}
