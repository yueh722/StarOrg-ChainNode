package client

import (
	"chain/lib"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/rpc/common/notification"
	"chain/utils"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/ethereum/go-ethereum/rpc"
)

type SetupClient struct {
	client          *rpc.Client
	logStore        queuestore.QueueStore
	cfg             *ClientConfig
	nodeRuntime     lib.NodeRuntime
	mux             sync.Mutex
	handledStopTime time.Time
}

var router = map[common.CmdType]func(*SetupClient, *common.CommandEnvelope) error{
	common.CmdInitializeChainNode: (*SetupClient).handleInitChainNodeCmd,
	common.CmdUpdateChainCfg:      (*SetupClient).handleUpdateChainCfgCmd,
	common.CmdControlNode:         (*SetupClient).handleControlNodeCmd,
	common.CmdPurgeChain:          (*SetupClient).handlePurgeChainCmd,
	common.CmdAddValidator:        (*SetupClient).HandleAddValidator,
	common.CmdPing:                (*SetupClient).HandlePing,
}

func newSetupClient(client *rpc.Client, cfg *ClientConfig) *SetupClient {
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

func (c *SetupClient) GetCurrentPeer() *lib.Peer {
	peerInfo, err := lib.GetPeerInfo(c.cfg.HomeDir)
	if err != nil {
		return &lib.Peer{}
	}
	return peerInfo
}

func (c *SetupClient) SubscribeCommand() <-chan error {

	var errChan = make(chan error, 5)

	ctx, cancel := context.WithCancel(context.Background())
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmdChan := make(chan common.CommandEnvelope, 1000)

	statusNode, err := BuildStatusNode(c.cfg, c.nodeRuntime)
	if err != nil {
		log.Infof("build status node error: %v", err)
		errChan <- err
		close(cmdChan)
		return errChan
	}

	sub, err := c.client.Subscribe(
		ctx,
		common.SetupNamespace{}.GetNamesapceString(),
		cmdChan,
		common.SetupNamespace{}.GetSubscriptionName(),
		common.DelegatorInfo{
			App:        c.cfg.AppName,
			StatusNode: *statusNode,
			DeviceID:   c.cfg.DeviceID,
		},
	)

	if err != nil {
		log.Infof("subscribe command error: %v", err)
		errChan <- err
		close(cmdChan)
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
				log.Infof("subscription error, server might closed connection. err: %v", errChan)

				ConnNotificationMgr().Notify(notification.Message{
					Type:    notification.MsgTypeOffline,
					Payload: c.cfg.ServerAddr,
					Time:    time.Now().UnixNano(),
				})

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

	c.cfg.Initialized = true

	//reload nodeInfo
	c.cfg.NodeInfo, err = lib.GetNodeInfo(c.cfg.HomeDir)
	if err != nil {
		log.Error("[handleInitChainNodeCmd] failed to reload node info upon initialized chain node, err:", err)
		return err
	}

	log.Infof("[handleInitChainNodeCmd] succeed. nodeID: %s, nodeName: %s", c.NodeID(), c.NodeName())

	statusNode, err := BuildStatusNode(c.cfg, nil)
	if err != nil {
		log.Info("[handleInitChainNodeCmd] failed to build status node upon initialize node, err:", err)
		return err
	}

	// use go-routine sending complete notification to prevent deadlock of master.
	go c.CompleteInitNodeCmd(statusNode)

	return nil
}

func (c *SetupClient) CompleteInitNodeCmd(statusNode *common.StatusNode) {

	if err := c.client.Call(nil, common.GetMethodName("InitializationComplete"), c.NodeID(), statusNode); err != nil {
		log.Info("failed to call InitializationComplete, err:", err)
	}
}

// update chain config handler
func (c *SetupClient) handleUpdateChainCfgCmd(cmd *common.CommandEnvelope) error {

	arg := common.SetupUpdateChainCfg{}
	err := cmtjson.Unmarshal(cmd.Payload, &arg)
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
	err := cmtjson.Unmarshal(cmd.Payload, &arg)
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
	case common.NodeReset:
		err = c.handleResetNode()
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
func (c *SetupClient) handlePurgeChainCmd(cmd *common.CommandEnvelope) error {

	log.Info("[handlePurgeChainCmd] prepare to reset chain")
	defer log.Info("[handlePurgeChainCmd] reset chain complete")

	if c.nodeRuntime != nil {
		if err := c.nodeRuntime.StopNode(); err != nil {
			log.Info("[handlePurgeChainCmd] failed to stop node, err:", err)
		}
	} else {
		if err := lib.KillNode(c.cfg.NodeInfo.NodeID); err != nil {
			log.Info("[handlePurgeChainCmd] failed to kill node, err:", err)
		}
	}

	//wait process exit after it received signal
	time.Sleep(2 * time.Second)

	//remove node base directory
	nodeInfo, err := lib.PurgeChainNode(c.cfg.HomeDir, c.cfg.NodeInfo.NodeID, c.cfg.NodeInfo.NodeName)
	if err != nil {
		return fmt.Errorf("failed to remove node base directory, err: %v", err)
	}

	c.cfg.NodeInfo = nodeInfo
	c.cfg.Initialized = false

	return nil
}

func (c *SetupClient) reportCometbftStatus(catalog string, exit bool, err error) {

	var (
		message string
		status  string
	)
	if exit && err != nil {
		message = err.Error()
	}

	if !exit {
		status = "started"
	} else {
		status = "stopped"
	}

	statusProcess := &common.StatusCometbftProcess{
		Catalog:  catalog,
		NodeID:   c.NodeID(),
		Status:   status,
		Error:    message,
		IssuedAt: utils.NowString(),
	}

	if err := c.client.Call(nil, common.GetMethodName("ReportCometbftProcessStatus"), statusProcess); err != nil {
		log.Info("failed to call ReportCometbftProcessStatus, err:", err)
	}

	if c.cfg.RestartOnNodeExited && c.handledStopTime.IsZero() {
		if err := c.handleRestartNode(); err != nil {
			log.Infof("[reportCometbftStatus] fail to start node after exit, err: %v", err)
		}
	}
}

func (c *SetupClient) buildNodeArg() lib.NodeArg {
	return lib.NodeArg{
		LogFilePath:    c.cfg.getLogPath(),
		LogStore:       c.logStore,
		NodeInfo:       c.cfg.NodeInfo,
		BinPath:        c.cfg.getBinPath(),
		NodeBaseDir:    c.cfg.NodeBaseDir,
		AppLogger:      nil,
		NodeLogger:     nil,
		NodeNotifyFunc: func(exit bool, err error) { c.reportCometbftStatus("node", exit, err) },
		AppNotifyFunc:  func(exit bool, err error) { c.reportCometbftStatus("app", exit, err) },
	}
}

// start node & app
func (c *SetupClient) handleStartNode() error {
	if c.nodeRuntime != nil && c.nodeRuntime.IsRunning() {
		return errors.New("node or app already running")
	}

	if !c.cfg.Initialized {
		return errors.New("node not initialized")
	}

	//clear time of last handledStop
	c.handledStopTime = time.Time{}
	nodeArg := c.buildNodeArg()
	runtime := lib.StartNode(c.NodeID(), lib.WithNodeArgs(nodeArg))
	err := runtime.Error()
	if err != nil {
		return err
	}

	c.nodeRuntime = runtime

	//write process details into file
	if err := WriteRuntimeData(c.cfg.getNodeRuntimeFilePath(), runtime); err != nil {
		log.Infof("[handleStartNode] failed to write process information of runtime , err: %v", err)
	}

	return nil
}

// add validator
func (c *SetupClient) HandleAddValidator(cmd *common.CommandEnvelope) error {

	arg := common.SetupAddValidator{}
	err := cmtjson.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	if isvalidator, _ := lib.IsNodeOriginalValidator(c.cfg.NodeBaseDir); !isvalidator {
		return errors.New("Ignore add validator from current node. current node is not an original validator, which does not have right to commit transaction.")
	}

	if err := lib.SendAddValidatorRequest(c.GetCurrentPeer().WebServiceAddr(), &arg.AddValidatorReq); err != nil {
		return err
	}

	return nil
}

func (c *SetupClient) HandlePing(cmd *common.CommandEnvelope) error {
	arg := common.SetupPing{}
	err := cmtjson.Unmarshal(cmd.Payload, &arg)
	if err != nil {
		return err
	}

	if arg.NodeID != c.NodeID() {
		return fmt.Errorf("invalid ping with node, nodeID: %v", arg.NodeID)
	}

	return nil
}

// stop node & app
func (c *SetupClient) handleStopNode() error {
	c.handledStopTime = time.Now()
	_ = WriteRuntimeData(c.cfg.getNodeRuntimeFilePath(), nil)

	err := lib.StopNode(c.nodeRuntime)
	if err != nil {
		return err
	}

	c.nodeRuntime = nil
	return nil
}

// restart node & app
func (c *SetupClient) handleRestartNode() error {
	if err := c.handleStopNode(); err != nil {
		return err
	}

	return c.handleStartNode()
}

// reset state of one node to uninitialized
func (c *SetupClient) handleResetNode() error {

	log.Info("[handleResetNode] prepare to reset node")
	defer log.Info("[handleResetNode] reset node complete")

	if err := lib.StopNode(c.nodeRuntime); err != nil {
		return fmt.Errorf("[handleResetNode] fail to stop node before reset node, err: %v", err)
	}

	if err := lib.ResetState(c.cfg.NodeBaseDir); err != nil {
		return fmt.Errorf("[handleResetNode] failed to reset state, err: %v", err)
	}

	if err := c.handleStartNode(); err != nil {
		return fmt.Errorf("[handleResetNode] fail to start node after reset node, err: %v", err)
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

	err := c.client.Call(nil, common.GetMethodName("ReportLog"), *event)
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
		message string
	)

	switch cmd.Cmd {
	case common.CmdPing:
		return
	case common.CmdControlNode:
		arg := common.SetupControlNode{}
		_ = cmtjson.Unmarshal(cmd.Payload, &arg)
		message = string(arg.Action)
	}

	if cmd.Cmd == common.CmdPing {
		//ignore sending result of command ping back
		return
	}

	if cmdErr != nil {
		errText = cmdErr.Error()
		status = commandlog.CmdFailed
	} else {
		status = commandlog.CmdSuccess
	}

	err := c.client.Call(nil, common.GetMethodName("ReportCommandResult"), commandlog.CommandResult{
		CommandID:  cmd.CommandID,
		Cmd:        cmd.Cmd,
		NodeID:     cmd.NodeID,
		Status:     status,
		Message:    message,
		Error:      errText,
		StartedAt:  cmd.IssuedAt,
		FinishedAt: utils.NowString(),
	})
	if err != nil {
		log.Println("failed to report command result:", err)
	}
}
