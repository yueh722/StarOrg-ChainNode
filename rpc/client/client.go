package client

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	Conn         *rpc.Client
	SetupClient  *SetupClient
	StatusClient *StatusClient
}

type ClientConfig struct {
	AppName    string
	ServerAddr string
}

func New(cfg ClientConfig) (*Client, error) {

	client, err := rpc.DialContext(context.Background(), cfg.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect %s; err: %v", cfg.ServerAddr, err)
	}

	setupClient := newSetupClient(client)
	statusClient := newStatusClient(client)

	return &Client{
		Conn:         client,
		SetupClient:  setupClient,
		StatusClient: statusClient,
	}, nil
}
