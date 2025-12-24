package client

import "github.com/ethereum/go-ethereum/rpc"

type StatusClient struct {
	client *rpc.Client
}

func newStatusClient(client *rpc.Client) *StatusClient {
	return &StatusClient{client: client}
}
