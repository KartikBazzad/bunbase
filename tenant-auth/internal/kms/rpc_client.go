package kms

import (
	"github.com/kartikbazzad/bunbase/pkg/kmsrpc"
)

// RPCClient wraps pkg/kmsrpc.Client and implements ClientInterface (string-based GetSecret/PutSecret).
type RPCClient struct {
	*kmsrpc.Client
}

// NewRPCClient creates a KMS client that uses the bun-kms TCP RPC. addr is e.g. "bunkms:9092".
func NewRPCClient(addr string) *RPCClient {
	return &RPCClient{Client: kmsrpc.New(addr)}
}

// GetSecret implements ClientInterface.
func (c *RPCClient) GetSecret(name string) (string, error) {
	b, err := c.Client.GetSecret(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PutSecret implements ClientInterface.
func (c *RPCClient) PutSecret(name, value string) error {
	return c.Client.PutSecret(name, []byte(value))
}
