package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	Conn         *rpc.Client
	Cfg          ClientConfig
	SetupClient  *SetupClient
	StatusClient *StatusClient
	mux          sync.Mutex
}

type ClientConfig struct {
	AppName    string
	ServerAddr string
}

func New(cfg ClientConfig) (*Client, error) {

	var (
		client *rpc.Client
		err    error
	)

	for {
		client, err = rpc.DialContext(context.Background(), cfg.ServerAddr)
		if err != nil {
			fmt.Println("Failed to connect rpc server", cfg.ServerAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}

	setupClient := newSetupClient(client)
	statusClient := newStatusClient(client)

	return &Client{
		Conn:         client,
		Cfg:          cfg,
		SetupClient:  setupClient,
		StatusClient: statusClient,
	}, nil
}

func (c *Client) Reconnect() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for {
		client, err := rpc.DialContext(context.Background(), c.Cfg.ServerAddr)
		if err != nil {
			log.Printf("Failed to connect %s; err: %v", c.Cfg.ServerAddr, err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.Conn = client
		c.SetupClient.Update(client)
		c.StatusClient.Update(client)
		log.Println("Reconnected successfully:", c.Cfg.ServerAddr)
		break
	}
}
