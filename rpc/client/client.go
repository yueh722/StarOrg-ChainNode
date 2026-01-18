package client

import (
	"chain/lib"
	log "chain/lib/zlog"
	"chain/rpc/security"
	"chain/utils"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
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
	HomeDir     string
	BinDir      string
	NodeBaseDir string
	AppDataDir  string
	LogDir      string
	NodeInfo    *lib.NodeInfo
}

func (c ClientConfig) getLogPath() string {
	return utils.TryCreateDir(c.LogDir)
}

func (c ClientConfig) getBinPath() string {
	return utils.TryCreateDir(c.BinDir)
}

func (c ClientConfig) getChainlogFilePath() string {
	return filepath.Join(c.getLogPath(), "chainlog.db")
}

func (c ClientConfig) getAppBinFile() string {
	return filepath.Join(c.getBinPath(), lib.AppFileName)
}

func (c ClientConfig) getNodeBinFile() string {
	return filepath.Join(c.getBinPath(), lib.NodeFileName)
}

func MakeRpcHeader(clientID string) http.Header {
	clientID = fmt.Sprintf("%s:%v", clientID, security.RandString(8))

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", security.GenerateToken(clientID, security.SharedSecret)))
	header.Set("X-Client-ID", clientID)

	return header
}

func New(cfg ClientConfig) (*Client, error) {

	var (
		client *rpc.Client
		err    error
	)

	utils.TryCreateDir(cfg.HomeDir)

	for {
		client, err = Connect(&cfg.ServerAddr, MakeRpcHeader(cfg.AppName))
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("server connected successfully: ", cfg.ServerAddr)
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
		client, err := Connect(&c.Cfg.ServerAddr, MakeRpcHeader(c.Cfg.AppName))
		if err != nil {
			log.Println(err)
			time.Sleep(5 * time.Second)
			continue
		}
		c.Conn = client
		c.SetupClient.Update(client)
		c.StatusClient.Update(client)
		log.Println("Reconnected successfully: ", c.Cfg.ServerAddr)
		break
	}
}

// Connect will update rawurl to the real url that connected.
func Connect(rawurl *string, header http.Header) (*rpc.Client, error) {

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

			client, err = rpc.DialOptions(context.Background(), addr, rpc.WithWebsocketDialer(dialer), rpc.WithHeaders(header))
		} else {
			client, err = rpc.DialOptions(context.Background(), addr, rpc.WithHeaders(header))
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
