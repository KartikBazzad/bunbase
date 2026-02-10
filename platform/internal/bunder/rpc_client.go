package bunder

import (
	"github.com/kartikbazzad/bunbase/pkg/bunderrpc"
)

// Ensure rpcClientProxy implements Proxy.
var _ Proxy = (*rpcClientProxy)(nil)

// rpcClientProxy wraps pkg/bunderrpc.Client to implement bunder.Proxy.
type rpcClientProxy struct {
	*bunderrpc.Client
}

// NewRPCClient returns a Proxy that uses the shared bunder RPC client (TCP).
func NewRPCClient(addr string) *rpcClientProxy {
	return &rpcClientProxy{Client: bunderrpc.New(addr)}
}

// ProxyRequest implements Proxy by delegating to bunderrpc.Client.
func (r *rpcClientProxy) ProxyRequest(method, projectID, path string, body []byte) (int, []byte, error) {
	return r.Client.ProxyRequest(method, projectID, path, body)
}
