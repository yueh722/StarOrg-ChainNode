package client

import (
	"chain/lib"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
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
	common.CmdResetChain:          (*SetupClient).handleResetChainCmd,
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

	ctx, cancel := context.WithCancel(context.Background())
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmdChan := make(chan common.CommandEnvelope, 1000)

	sub, err := c.client.Subscribe(
		ctx,
		common.SetupNamespace{}.GetNamesapceString(),
		cmdChan,
		common.SetupNamespace{}.GetSubscriptionName(),
		common.DelegatorInfo{
			App: c.cfg.AppName,
			StatusNode: common.StatusNode{
				NodeInfo:    *c.cfg.NodeInfo,
				Healthy:     lib.CheckNodeOnline(c.cfg.NodeInfo.RPCAddr),
				LastCheck:   utils.NowString(),
				StartedTime: lib.GetNodeCreatedTime(c.nodeRuntime),
			},
		},
	)

	if err != nil {
		log.Error("SubscribeCommand error: ", err)
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
				log.Println("recived command: ", cmdEnvelope.Cmd)
				err := c.handleCmd(&cmdEnvelope)
				if err != nil {
					log.Println("failed to handle cmd: ", err)
				}
			case errChan := <-sub.Err():
				log.Println("subscription Err: ", errChan)
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

	if fn, ok := router[cmd.Cmd]; ok {
		err := fn(c, cmd)
		c.reportCommandResult(cmd, err)
		return err
	}

	return fmt.Errorf("unknown cmd: %v", cmd.Cmd)
}

// init node handler
func (c *SetupClient) handleInitChainNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupInitializeChainNode{}
	err := cmtjson.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.InitializeChainNode(
		&arg.GenesisDoc,
		arg.NodeArray,
		arg.NodeID,
		c.cfg.HomeDir,
	)
	if err != nil {
		nodeName := arg.NodeArray.GetPeerByNodeId(arg.NodeID).NodeName
		log.Printf("[handleInitChainNodeCmd] failed to init chain node id: %s, node name: %s, err: %v", cmd.NodeID, nodeName, err)
		return err
	}

	//reload nodeInfo
	c.cfg.NodeInfo, err = lib.GetNodeInfo(c.cfg.HomeDir)
	if err != nil {
		log.Error("[handleInitChainNodeCmd] failed to reload node info upon initialized chain node, err:", err)
		return err
	}

	log.Infof("[handleInitChainNodeCmd] succeed. nodeID: %s, nodeName: %s", c.NodeID(), c.NodeName())

	return nil
}

// update chain config handler
func (c *SetupClient) handleUpdateChainCfgCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupUpdateChainCfg{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	err = lib.UpdateChainCfg(
		arg.NodeArray,
		c.cfg.HomeDir,
		c.cfg.NodeInfo.NodeID,
	)
	if err != nil {
		log.Errorf("failed to add chain nodeID: %s, nodeName: %s, err: %v", cmd.NodeID, c.NodeName(), err)
		return err
	}

	log.Infof("[handleUpdateChainCfgCmd] succeed. nodeID: %s, nodeName: %s", c.NodeID(), c.NodeName())

	return nil
}

// control node handler
func (c *SetupClient) handleControlNodeCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupControlNode{}
	err := json.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	//create logStore for receiving error zlog of node and app
	//file: log/chaindb.log
	if err = c.initLogStore(c.cfg.getChainlogFilePath()); err != nil {
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
		return fmt.Errorf("failed to %s chain node id: %s, node name: %s, err: %v", arg.Action, cmd.NodeID, c.NodeName(), err)
	}

	log.Infof("[handleControlNodeCmd] succeed. action: %s, nodeID: %s, nodeName: %s", arg.Action, c.NodeID(), c.NodeName())

	return nil
}

// reset chain handler
func (c *SetupClient) handleResetChainCmd(cmd *common.CommandEnvelope) error {

	if c.nodeRuntime != nil {
		if err := c.nodeRuntime.KillNode(); err != nil {
			log.Error("[handleResetChainCmd] failed to kill node, err:", err)
		}
	} else {
		//kill all node and app processes
		if err := utils.KillProcessByProcNameAndCmdline(lib.AppFileName, c.cfg.NodeInfo.NodeID); err != nil {
			log.Info("[handleResetChainCmd] failed to kill app process, err: ", err)
		}

		if err := utils.KillProcessByProcNameAndCmdline(lib.NodeFileName, c.cfg.NodeInfo.NodeID); err != nil {
			log.Info("[handleResetChainCmd] failed to kill node process, err: ", err)
		}
	}

	time.Sleep(2 * time.Second)

	//remove node base directory
	nodeInfo, err := lib.ResetChainNode(c.cfg.HomeDir, c.cfg.NodeInfo.NodeID, c.cfg.NodeInfo.NodeName)
	if err != nil {
		return fmt.Errorf("failed to remove node base directory, err: %v", err)
	}

	c.cfg.NodeInfo = nodeInfo

	return nil
}

// start node & app
func (c *SetupClient) handleStartNode() error {
	if c.nodeRuntime != nil && c.nodeRuntime.IsRunning() {
		return errors.New("node or app already running")
	}

	if !lib.IsNodeInitialized(c.cfg.NodeInfo) {
		return errors.New("node not initialized")
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

// stop node & app
func (c *SetupClient) handleStopNode() error {
	err := lib.StopNode(c.nodeRuntime)
	if err == nil {
		c.nodeRuntime = nil
		return nil
	}

	return err
}

// restart node & app
func (c *SetupClient) handleRestartNode() error {
	if err := c.handleStopNode(); err != nil {
		return err
	}

	return c.handleStartNode()
}

func (c *SetupClient) buildNodeArg() lib.NodeArg {
	return lib.NodeArg{
		LogFilePath: c.cfg.getLogPath(),
		LogStore:    c.logStore,
		NodeInfo:    c.cfg.NodeInfo,
		BinPath:     c.cfg.getBinPath(),
		NodeBaseDir: c.cfg.NodeBaseDir,
		AppLogger:   nil,
		NodeLogger:  nil,
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

	ch := c.logStore.GetBroker().Subscribe(256)
	//subscribe data from store
	go func() {
		defer c.logStore.GetBroker().Unsubscribe(ch)
		for {
			event, ok := <-ch
			if !ok {
				return
			}

			log.Println("log event:", event)
			_ = c.reportLog(&event)
		}
	}()

	return nil
}

func (c *SetupClient) reportLog(event *queuestore.PersistedEvent) error {

	method := common.MakeMethodName(common.NodeLogNamespace{}.GetNamesapceString(), "ReportLog")
	err := c.client.Call(nil, method, *event)
	if err != nil {
		return err
	}
	return nil
}

// report result of command to NodeMaster
func (c *SetupClient) reportCommandResult(cmd *common.CommandEnvelope, cmdErr error) {

	var (
		errText string
		status  commandlog.CommandStatus
	)

	if cmdErr != nil {
		errText = cmdErr.Error()
		status = commandlog.CmdFailed
	} else {
		status = commandlog.CmdSuccess
	}

	method := common.MakeMethodName(common.SetupNamespace{}.GetNamesapceString(), "ReportCommandResult")
	err := c.client.Call(nil, method, commandlog.CommandResult{
		CommandID:  cmd.CommandID,
		Cmd:        cmd.Cmd,
		NodeID:     cmd.NodeID,
		Status:     status,
		Message:    "",
		Error:      errText,
		ReportedAt: time.Now().UnixNano(),
		StartedAt:  cmd.IssuedAt,
		FinishedAt: time.Now().UnixNano(),
		Output:     nil,
	})
	if err != nil {
		log.Println("", err)
	}
}
