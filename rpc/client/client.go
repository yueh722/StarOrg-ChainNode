package client

import (
	"chain/lib"
	log "chain/lib/zlog"
	"chain/rpc/common"
	"chain/rpc/common/notification"
	"chain/rpc/security"
	"chain/utils"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn          *rpc.Client
	Cfg           *ClientConfig
	SetupClient   *SetupClient
	StatusClient  *StatusClient
	TxClient      *LiveTxClient
	mux           sync.Mutex
	OnReconnected func(*Client)
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
	Initialized bool
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

func (c ClientConfig) getNodeDBPath() string {
	return filepath.Join(c.NodeBaseDir, "data")
}

func (c ClientConfig) getNodeRuntimeFilePath() string {
	return filepath.Join(c.HomeDir, "runtime.json")
}

var connectionNotifier = notification.NewManager()

func ConnNotificationMgr() *notification.Manager {
	return connectionNotifier
}

func MakeRpcHeader(clientID string) http.Header {
	clientID = fmt.Sprintf("%s:%v", clientID, security.RandString(8))

	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("Bearer %s", security.GenerateToken(clientID, security.SharedSecret)))
	header.Set("X-Client-ID", clientID)

	return header
}

func BuildStartCmd(nodeID string) *common.CommandEnvelope {

	payloadStruct := common.SetupControlNode{
		NodeID: nodeID,
		Action: common.NodeStart,
	}
	payload, _ := cmtjson.Marshal(payloadStruct)

	return &common.CommandEnvelope{
		CommandID: utils.UUID(),
		Cmd:       common.CmdControlNode,
		NodeID:    nodeID,
		Payload:   payload,
		IssuedAt:  time.Now().Unix(),
	}
}

func New(cfg *ClientConfig) (*Client, error) {

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
	liveTxClient := newLiveTxClient(client, cfg)

	//check last process info of node runtime
	if lastRuntime, _ := LoadLastNodeInfo(cfg.getNodeRuntimeFilePath()); lastRuntime != nil {
		if lastRuntime.Node.Pid != 0 && lastRuntime.App.Pid != 0 {
			// node and app might exit unexpected, so start node automatically.
			time.Sleep(2 * time.Second)
			_ = setupClient.handleControlNodeCmd(BuildStartCmd(cfg.NodeInfo.NodeID))
		}
	}

	return &Client{
		Conn:         client,
		Cfg:          cfg,
		SetupClient:  setupClient,
		StatusClient: statusClient,
		TxClient:     liveTxClient,
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
		c.TxClient.Update(client)
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

func BuildStatusNode(cfg *ClientConfig, nr lib.NodeRuntime) (*common.StatusNode, error) {

	ips, err := utils.GetAllIPAddresses()
	if err != nil {
		log.Fatalf("failed to get IP addresses, %v", err)
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname, %v", err)
		return nil, err
	}

	nodeStartTime := ""
	if nr != nil {
		nodeStartTime = nr.CreatedTime()
	} else {
		nodeStartTime = utils.GetProcessCreatedTime(lib.NodeFileName, lib.AppFileName)
	}

	return &common.StatusNode{
		NodeInfo: *cfg.NodeInfo,
		HostInfo: common.HostInfo{
			HostName: hostname,
			IpAddrs:  ips,
		},
		Healthy:     lib.CheckNodeOnline(cfg.NodeInfo.RPCAddr),
		LastCheck:   utils.NowString(),
		StartedTime: nodeStartTime,
		Initialized: cfg.Initialized,
	}, nil
}

type RuntimeProcInfo struct {
	App  lib.ProcInfo
	Node lib.ProcInfo
	Hash string
}

func (r *RuntimeProcInfo) computeHash() (string, error) {
	originalHash := r.Hash // Save original hash
	defer func() {
		r.Hash = originalHash
	}()

	r.Hash = ""               // Clear the hash field for computing
	b, err := json.Marshal(r) // Marshal entire struct with empty Hash
	if err != nil {
		return "", err
	}

	return utils.Sum256(b), nil
}

func (r *RuntimeProcInfo) Valid() error {

	expectedHash, err := r.computeHash()
	if err != nil {
		return err
	}

	if r.Hash != expectedHash {
		return fmt.Errorf("hash validation failed")
	}

	return nil
}

func (r *RuntimeProcInfo) Marshal() ([]byte, error) {
	hash, err := r.computeHash()
	if err != nil {
		return nil, err
	}

	r.Hash = hash
	return json.MarshalIndent(r, "", "  ")
}

func WriteRuntimeData(dataFile string, nr lib.NodeRuntime) error {

	if nr == nil {
		// delete runtime data
		if err := os.Remove(dataFile); err != nil {
			return fmt.Errorf("couldn't remove file that incldes runtime data: %w", err)
		}
		return nil
	}

	procInfo := &RuntimeProcInfo{
		App:  nr.GetAppInfo(),
		Node: nr.GetNodeInfo(),
		Hash: "",
	}

	data, err := procInfo.Marshal()
	if err != nil {
		return err
	}

	if err := os.WriteFile(dataFile, data, os.ModePerm); err != nil {
		return fmt.Errorf("write proc info of runtime to file failed: %v", err)
	}

	return nil
}

func LoadLastNodeInfo(dataFile string) (*RuntimeProcInfo, error) {

	procInfo := RuntimeProcInfo{}
	err := utils.AnyFromFile(dataFile, &procInfo)
	if err != nil {
		return nil, err
	}

	err = procInfo.Valid()
	if err != nil {
		_ = os.Remove(dataFile)
		return nil, err
	}

	return &procInfo, nil
}
