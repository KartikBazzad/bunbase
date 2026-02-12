package bundoc

import (
	"github.com/kartikbazzad/bunbase/pkg/bundocrpc"
)

// Ensure rpcClientProxy implements Proxy.
var _ Proxy = (*rpcClientProxy)(nil)

// rpcClientProxy wraps pkg/bundocrpc.Client to implement bundoc.Proxy.
type rpcClientProxy struct {
	*bundocrpc.Client
}

// NewRPCClient returns a Proxy that uses the shared bundoc RPC client (TCP).
func NewRPCClient(addr string) *rpcClientProxy {
	return &rpcClientProxy{Client: bundocrpc.New(addr)}
}

// ProxyRequest implements Proxy by delegating to bundocrpc.Client.
// Note: RPC transport doesn't currently support headers, so headers are ignored.
func (r *rpcClientProxy) ProxyRequest(method, projectID, path string, body []byte, headers map[string]string) (int, []byte, error) {
	// RPC transport doesn't support headers yet - can be added later if needed
	return r.Client.ProxyRequest(method, projectID, path, body)
}
