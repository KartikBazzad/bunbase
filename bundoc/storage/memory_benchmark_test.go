package storage

import (
	"fmt"
	"testing"
)

func BenchmarkDocumentSerialize(b *testing.B) {
	doc := make(Document)
	doc["_id"] = "1234567890"
	for i := 0; i < 1000; i++ {
		doc[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := doc.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocumentDeserialize(b *testing.B) {
	doc := make(Document)
	doc["_id"] = "1234567890"
	for i := 0; i < 1000; i++ {
		doc[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}
	data, _ := doc.Serialize()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := DeserializeDocument(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDocumentClone(b *testing.B) {
	doc := make(Document)
	doc["_id"] = "1234567890"
	for i := 0; i < 1000; i++ {
		doc[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = doc.Clone()
	}
}
