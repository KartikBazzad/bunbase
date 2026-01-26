package wal

import (
	"encoding/binary"
	"hash/crc32"

	"github.com/kartikbazzad/docdb/internal/types"
)

var byteOrder = binary.LittleEndian

func EncodeRecord(txID, dbID, docID uint64, opType types.OperationType, payload []byte) ([]byte, error) {
	payloadLen := uint32(len(payload))
	if uint32(payloadLen) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	totalLen := RecordOverhead + uint64(payloadLen)
	buf := make([]byte, totalLen)

	offset := 0

	byteOrder.PutUint64(buf[offset:], totalLen)
	offset += RecordLenSize

	byteOrder.PutUint64(buf[offset:], txID)
	offset += TxIDSize

	byteOrder.PutUint64(buf[offset:], dbID)
	offset += DBIDSize

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
	if len(data) < RecordOverhead {
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

	opType := types.OperationType(data[offset])
	offset += OpTypeSize

	docID := byteOrder.Uint64(data[offset:])
	offset += DocIDSize

	payloadLen := byteOrder.Uint32(data[offset:])
	offset += PayloadLenSize

	var payload []byte
	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		copy(payload, data[offset:offset+int(payloadLen)])
	}

	return &types.WALRecord{
		Length:     recordLen,
		TxID:       txID,
		DBID:       dbID,
		OpType:     opType,
		DocID:      docID,
		PayloadLen: payloadLen,
		Payload:    payload,
		CRC:        storedCRC,
	}, nil
}
