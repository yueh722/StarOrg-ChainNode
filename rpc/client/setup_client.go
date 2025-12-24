package client

import "github.com/ethereum/go-ethereum/rpc"

type SetupClient struct {
	client *rpc.Client
}

func newSetupClient(client *rpc.Client) *SetupClient {
	return &SetupClient{client: client}
}
