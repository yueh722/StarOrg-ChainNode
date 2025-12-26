package client

import (
	"chain/lib"
	"context"
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
	AppName     string
	ServerAddr  string
	NodeBaseDir string
	NodeInfo    *lib.NodeInfo
}

func New(cfg ClientConfig) (*Client, error) {

	var (
		client *rpc.Client
		err    error
	)

	for {
		client, err = rpc.DialContext(context.Background(), cfg.ServerAddr)
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("server connected successfully:", cfg.ServerAddr)
		break
	}

	setupClient := newSetupClient(client, cfg)
	statusClient := newStatusClient(client, cfg)

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
			log.Println(err)
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
