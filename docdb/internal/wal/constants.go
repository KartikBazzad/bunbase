package wal

const (
	RecordLenSize     = 8
	TxIDSize          = 8
	DBIDSize          = 8
	CollectionLenSize = 2 // v0.2: collection name length (0-65535 bytes)
	OpTypeSize        = 1
	DocIDSize         = 8
	PayloadLenSize    = 4
	CRCSize           = 4

	// v0.4 format: LSN and PayloadCRC
	LSNSize        = 8
	PayloadCRCSize = 4

	// v0.1 format (no collection field)
	HeaderSizeV1     = RecordLenSize + TxIDSize + DBIDSize + OpTypeSize + DocIDSize + PayloadLenSize
	RecordOverheadV1 = HeaderSizeV1 + CRCSize

	// v0.2 format (with collection field)
	// Note: Collection name length is variable, so HeaderSizeV2 is minimum
	HeaderSizeV2Min     = RecordLenSize + TxIDSize + DBIDSize + CollectionLenSize + OpTypeSize + DocIDSize + PayloadLenSize
	RecordOverheadV2Min = HeaderSizeV2Min + CRCSize

	// v0.4 format (partition WAL: LSN, PayloadCRC)
	HeaderSizeV4Min     = RecordLenSize + LSNSize + TxIDSize + DBIDSize + CollectionLenSize + OpTypeSize + DocIDSize + PayloadLenSize + PayloadCRCSize
	RecordOverheadV4Min = HeaderSizeV4Min + CRCSize

	// For backward compatibility, use v0.1 constants as defaults
	HeaderSize     = HeaderSizeV1
	RecordOverhead = RecordOverheadV1
)

const (
	MaxPayloadSize = 16 * 1024 * 1024
)
