package query

// SortDocuments sorts a slice of documents based on a field
// We use interface{} because storage.Document is basically map[string]interface{}
// But Go doesn't allow implicit slice conversion []T -> []interface{}
// So we might need to accept an interface that SortIterator can provide.
// Or we just accept interface{} and use reflection, OR better:
// The caller passes `sort.Slice` logic.
// But `SortDocuments` was called in `iterator.go`.
// Let's implement it to take `interface{}` and reflect? No, too slow.
// Let's define a Sortable interface or just use `sort.Slice` inside `SortIterator`?
// The problem is `iterator.go` is in `bundoc`, `SortDocuments` is in `query`.
// `iterator.go` knows about `storage.Document`.
// Maybe I should move `SortDocuments` logic into `iterator.go` or `bundoc/sort.go` instead of `query` package?
// The `query` package should probably just handle AST/Parsing.
// Execution logic (sorting docs) belongs closer to where docs are defined or in a `execution` package.
// For now, I will put `SortDocuments` in `bundoc/internal/query` but declare it accepts `[]map[string]interface{}`?
// No, `[]storage.Document` cannot be passed as `[]map[...]`.
// I will implement `SortDocuments` in `query` package but using a wrapper type or just generic implementation?
// Go generics! `SortDocuments[T ~map[string]interface{}](docs []T ...)`
// But I might be on older Go? Project says Go 1.25.6+. Generics are fine.

// Actually, simplest is to implement sorting inside `iterator.go` using `sort.Slice`.
// But I already committed `query.SortDocuments` call.
// So I will implement it in `query` package using sorting logic.
// I will need to cast `[]storage.Document` to something `query` understands?
// `query` package doesn't know `storage.Document`.
// I will make `SortDocuments` accept `interface{}` and assert, OR
// I will change `iterator.go` to use `sort.Slice` directly and remove `query.SortDocuments` call.
// Using `sort.Slice` in `iterator.go` is cleaner because it avoids dependency cycles and type matching issues.
// I will modify `iterator.go` to implement sorting inline.

func SortDocuments(docs interface{}, field string, desc bool) {
	// Placeholder if I decide to keep it
}
