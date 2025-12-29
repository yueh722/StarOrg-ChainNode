package client

import (
	"chain/lib"
	"context"
	"crypto/tls"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
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
		client, err = connect(&cfg.ServerAddr)
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
		client, err := connect(&c.Cfg.ServerAddr)
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

func connect(rawurl *string) (*rpc.Client, error) {

	serverAddrs := map[string]bool{}

	u, _ := url.Parse(*rawurl)

	//add ws url
	u.Scheme = "ws"
	serverAddrs[u.String()] = false

	//add wss url
	u.Scheme = "wss"
	serverAddrs[u.String()] = true

	var (
		err    error
		client *rpc.Client
	)

	for addr, tlsEnabled := range serverAddrs {

		log.Println("connecting to server:", addr)

		if tlsEnabled {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
			}

			dialer := websocket.Dialer{
				TLSClientConfig: tlsConfig,
			}

			client, err = rpc.DialOptions(context.Background(), addr, rpc.WithWebsocketDialer(dialer))
		} else {
			client, err = rpc.DialContext(context.Background(), addr)
		}

		if err != nil {
			log.Println(err)
			continue
		}

		*rawurl = addr
		return client, nil
	}

	return nil, err
}

func (c *Client) Register() error {

	//submit node info status

	return nil
}
