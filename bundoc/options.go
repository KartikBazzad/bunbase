package bundoc

// QueryOptions represents query options like sort, limit, skip, and field projection
type QueryOptions struct {
	SortField string
	SortDesc  bool
	Limit     int
	Skip      int
	// Fields, when non-nil and non-empty, limits returned documents to these top-level keys only
	Fields []string
}
