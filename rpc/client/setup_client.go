package client

import (
	"chain/lib"
	"chain/lib/queuestore"
	"chain/rpc/common"
	"chain/rpc/server"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type SetupClient struct {
	client      *rpc.Client
	logStore    queuestore.QueueStore
	cfg         ClientConfig
	nodeRuntime lib.NodeRuntime
	mux         sync.Mutex
}

var router = map[common.CmdType]func(*SetupClient, *common.CommandEnvelope) error{
	common.CmdInitializeChainNode: (*SetupClient).handleInitChainNodeCmd,
	common.CmdUpdateChainCfg:      (*SetupClient).handleUpdateChainCfgCmd,
	common.CmdControlNode:         (*SetupClient).handleControlNodeCmd,
}

func newSetupClient(client *rpc.Client, cfg ClientConfig) *SetupClient {
	return &SetupClient{
		client: client,
		cfg:    cfg,
	}
}

func (c *SetupClient) NodeID() string {
	return c.cfg.NodeInfo.NodeID
}

func (c *SetupClient) NodeName() string {
	return c.cfg.NodeInfo.NodeName
}

func (c *SetupClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *SetupClient) SubscribeCommand() <-chan error {

	var errChan = make(chan error)

	setupService := &server.SetupService{}
	//ctx, cancel := context.WithCancel(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmdChan := make(chan common.CommandEnvelope, 1000)

	sub, err := c.client.Subscribe(
		ctx,
		setupService.GetNamesapceString(),
		cmdChan,
		setupService.GetSubscriptionName(),
		c.cfg.NodeInfo,
	)

	if err != nil {
		return errChan
	}

	go func() {
		defer func() {
			sub.Unsubscribe()
			errChan <- rpc.ErrClientQuit
			close(cmdChan)
		}()

		exitSub := false
		for !exitSub {
			select {
			case cmdEnvelope := <-cmdChan:
				log.Printf("command: %v", cmdEnvelope.Payload)
				err := c.handleCmd(&cmdEnvelope)
				if err != nil {
					log.Printf("failed to handle cmd: %v", err)
				}
			case errChan := <-sub.Err():
				log.Printf("subscription Err: %v", errChan)
				exitSub = true
				break
			}
		}
	}()

	return errChan
}

func (c *SetupClient) handleCmd(cmd *common.CommandEnvelope) error {
	if cmd.NodeID != c.NodeID() {
		return fmt.Errorf("nodeID mismatch, ignore")
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	err := router[cmd.Cmd](c, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *SetupClient) handleInitChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupInitializeChainNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.InitializeChainNode(
		&arg.GenesisDoc,
		arg.NodeArray,
		arg.NodeID,
		c.cfg.NodeBaseDir,
	)
	if err != nil {
		nodeName := arg.NodeArray.GetPeerByNodeId(arg.NodeID).NodeName
		log.Printf("failed to init chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
		return err
	}

	//reload nodeInfo of cfg
	c.cfg.NodeInfo, err = lib.GetNodeInfo(c.cfg.NodeBaseDir)
	if err != nil {
		log.Println("failed to reload node info upon initialized chain node, err:", err)
		return err
	}

	return nil
}

func (c *SetupClient) handleUpdateChainCfgCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupUpdateChainCfg{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.UpdateChainCfg(
		arg.NodeArray,
		c.cfg.NodeBaseDir,
	)
	if err != nil {
		log.Printf("failed to add chain node id: %s, node name: %s, err: %v", cmd.NodeID, c.NodeName(), err)
		return err
	}

	return nil
}

func (c *SetupClient) handleControlNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupControlNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	//create logStore for receiving error log of node and app
	if err = c.initLogStore(filepath.Join(c.cfg.NodeBaseDir, "log", "chainlog.db")); err != nil {
		return err
	}

	switch arg.Action {
	case common.NodeStart:
		err = c.handleStartNode()
	case common.NodeStop:
		err = c.handleStopNode()
	case common.NodeRestart:
		err = c.handleRestartNode()
	default:
		err = fmt.Errorf("invalid action")
	}

	if err != nil {
		return fmt.Errorf("failed to %s chain node id: %s, node name: %s, err: %v", arg.Action, cmd.NodeID, err)
	}

	return nil
}

func (c *SetupClient) handleStartNode() error {
	if c.nodeRuntime != nil && c.nodeRuntime.IsRunning() {
		return errors.New("node or app already running")
	}

	nodeArg := c.buildNodeArg()
	runtime := lib.StartNode(c.NodeID(), lib.WithNodeArgs(nodeArg))
	err := runtime.Error()
	if err != nil {
		return err
	}

	c.nodeRuntime = runtime
	return nil
}

func (c *SetupClient) handleStopNode() error {
	return lib.StopNode(c.nodeRuntime)
}

func (c *SetupClient) handleRestartNode() error {
	if err := c.handleStopNode(); err != nil {
		return err
	}

	return c.handleStartNode()
}

func (c *SetupClient) buildNodeArg() lib.NodeArg {
	return lib.NodeArg{
		LogFilePath: filepath.Join(c.cfg.NodeBaseDir, "log"),
		LogStore:    c.logStore,
		NodeInfo:    c.cfg.NodeInfo,
		BinPath:     filepath.Join(c.cfg.NodeBaseDir, "bin"),
		NodeBaseDir: c.cfg.NodeBaseDir,
	}
}

func (c *SetupClient) initLogStore(logFilePath string) error {

	if c.logStore != nil {
		return nil
	}

	logStore, err := queuestore.NewSQLiteStore(
		queuestore.SQLiteConfig{
			Path:        logFilePath,
			Migrate:     true,
			OpTimeout:   5 * time.Second,
			InitTimeout: 3 * time.Second,
		})
	if err != nil {
		return fmt.Errorf("failed to init core store upon initialize chain node, err: %v", err)
	}

	c.logStore = logStore

	return nil
}
