package client

import (
	"chain/lib"
	"chain/lib/queuestore"
	"chain/rpc/common"
	"chain/rpc/server"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type SetupClient struct {
	client   *rpc.Client
	logStore queuestore.QueueStore
	cfg      ClientConfig
}

var router = map[common.CmdType]func(SetupClient, *common.CommandEnvelope) error{
	common.CmdInitializeChainNode: SetupClient.handleInitChainNodeCmd,
	common.CmdAddChainNode:        SetupClient.handleAddChainNodeCmd,
	common.CmdControlNode:         SetupClient.handleControlNodeCmd,
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
		c.NodeID(),
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

func (c SetupClient) handleCmd(cmd *common.CommandEnvelope) error {
	if cmd.NodeID != c.NodeID() {
		return fmt.Errorf("nodeID mismatch, ignore")
	}

	err := router[cmd.Cmd](c, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c SetupClient) handleInitChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupInitializeChainNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.InitializeChainNode(
		arg.GenesisTime,
		arg.NodeArray,
		arg.ChainID,
		arg.NodeID,
		c.cfg.NodeBaseDir,
	)
	if err != nil {
		nodeName := lib.GetNodeNameByNodeId(arg.NodeID, arg.NodeArray)
		log.Printf("failed to init chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
		return err
	}

	//reload nodeInfo of cfg
	c.cfg.NodeInfo, err = lib.GetNodeInfo(c.cfg.NodeBaseDir)
	if err != nil {
		log.Println("failed to reload node info upon initialize chain node, err:", err)
	}

	return nil
}

func (c SetupClient) handleAddChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupAddChainNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.AddChainNode(
		arg.NodeArray,
		arg.GenesisDoc,
		arg.NodeID,
		c.cfg.NodeBaseDir,
	)
	if err != nil {
		nodeName := lib.GetNodeNameByNodeId(arg.NodeID, arg.NodeArray)
		log.Printf("failed to add chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
		return err
	}

	//reload nodeInfo of cfg
	c.cfg.NodeInfo, err = lib.GetNodeInfo(c.cfg.NodeBaseDir)
	if err != nil {
		log.Println("failed to reload node info upon initialize chain node, err:", err)
	}

	return nil
}

func (c SetupClient) handleControlNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupControlNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	//create logStore for receiving error log of node and app
	if err = c.initLogStore(filepath.Join(c.cfg.NodeBaseDir, "log", "chainlog.db")); err != nil {
		return err
	}

	type RunArg struct {
		fn  func(nodeID string, opts ...lib.CtlOption) error
		opt lib.CtlOption
	}

	nodeArg := lib.NodeArg{
		LogFilePath: filepath.Join(c.cfg.NodeBaseDir, "log"),
		LogStore:    c.logStore,
		NodeInfo:    c.cfg.NodeInfo,
		BinPath:     filepath.Join(c.cfg.NodeBaseDir, "bin"),
		NodeBaseDir: c.cfg.NodeBaseDir,
	}

	router := map[common.NodeAction]RunArg{
		common.NodeStart: {
			fn:  lib.StartNode,
			opt: lib.WithNodeArgs(nodeArg),
		},
		common.NodeStop: {
			fn: lib.StopNode,
		},
		common.NodeRestart: {
			fn:  lib.RestartNode,
			opt: lib.WithNodeArgs(nodeArg),
		},
	}

	runOption, ok := router[arg.Action]
	if !ok {
		return fmt.Errorf("invalid action: %v", arg.Action)
	}

	err = runOption.fn(arg.NodeID, runOption.opt)
	if err != nil {
		return fmt.Errorf("failed to %s chain node id: %s, node name: %s, err: %v", arg.Action, cmd.NodeID, err)
	}

	return nil
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
