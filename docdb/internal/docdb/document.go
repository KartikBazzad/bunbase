package docdb

import (
	"github.com/kartikbazzad/docdb/internal/types"
)

type Document struct {
	ID      uint64
	Payload []byte
}

func (d *Document) ToType() *types.Document {
	return &types.Document{
		ID:      d.ID,
		Payload: d.Payload,
	}
}

func FromType(doc *types.Document) *Document {
	return &Document{
		ID:      doc.ID,
		Payload: doc.Payload,
	}
}
