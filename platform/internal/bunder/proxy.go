package bunder

// Proxy is the interface for KV proxy (HTTP or RPC). KVHandler uses this so it can use either transport.
type Proxy interface {
	ProxyRequest(method, projectID, path string, body []byte) (int, []byte, error)
}
