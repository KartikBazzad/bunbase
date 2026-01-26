package wal

const (
	RecordLenSize  = 8
	TxIDSize       = 8
	DBIDSize       = 8
	OpTypeSize     = 1
	DocIDSize      = 8
	PayloadLenSize = 4
	CRCSize        = 4

	HeaderSize     = RecordLenSize + TxIDSize + DBIDSize + OpTypeSize + DocIDSize + PayloadLenSize
	RecordOverhead = HeaderSize + CRCSize
)

const (
	MaxPayloadSize = 16 * 1024 * 1024
)
