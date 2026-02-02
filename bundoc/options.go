package bundoc

// QueryOptions represents query options like sort, limit, skip
type QueryOptions struct {
	SortField string
	SortDesc  bool
	Limit     int
	Skip      int
}
