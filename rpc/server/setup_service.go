package server

import (
	"chain/lib"
	"chain/lib/cmdlogstore"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/rpc/server/command_manager"
	"chain/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pingcap/errors"
)

type SetupResult struct {
	Error error `json:"error"`
}

type MultiError struct {
	Errors map[string]error
}

func (m *MultiError) Error() string {
	errStr, err := json.MarshalIndent(m.Errors, "", "  ")
	if err != nil {
		return err.Error()
	}

	return string(errStr)
}

type nodeCommand struct {
	cmd     common.CmdType
	payload interface{}
	ctx     context.Context
}

// SetupService is designed for nodeproxy getting setup command dispatched from NodeMaster.
// Request from: NodeProxy
type SetupService struct {
	mu         sync.Mutex
	subs       map[string]*ClientSub
	stream     *StreamService
	cmdEvent   commandlog.CommandEventRepo
	chainState *ChainStateInternal
	cmdManager *command_manager.CmdManager
}

type ClientSub struct {
	Notifier   *rpc.Notifier
	NodeID     string
	StatusNode common.StatusNode
	SubID      rpc.ID
}

var chainStateFile = filepath.Join(utils.GetAppDataPath("SetupService"), "setup.json")

func init() {
	utils.TryCreateDir(filepath.Dir(chainStateFile))
}

func NewSetupService(store cmdlogstore.CommandLogStore, stream *StreamService) *SetupService {

	cmdEvent, err := commandlog.NewCommandRepo(store)
	if err != nil {
		log.Fatal(err)
	}

	return &SetupService{
		subs:       make(map[string]*ClientSub),
		cmdEvent:   cmdEvent,
		stream:     stream,
		chainState: LoadChainState(chainStateFile),
		cmdManager: command_manager.NewCmdManager(),
	}
}

func (s *SetupService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(common.SetupNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register chain service: %v", err)
	}

	return nil
}

func ReturnError(err error) error {
	return err
}

// RPC Method
// Fetch: "retrive chain state and node state"
func (s *SetupService) GetChainState() *ChainState {

	var nodes []common.StatusNode

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, clientSub := range s.subs {

		node := &clientSub.StatusNode
		node.Healthy = lib.CheckNodeOnline(node.RPCAddr)
		node.LastCheck = time.Now().UnixNano()

		nodes = append(nodes, *node)
	}

	if s.chainState.Initialized() {
		return &ChainState{
			Nodes:           nodes,
			ChainID:         s.chainState.GenesisDoc.ChainID,
			InitializedTime: s.chainState.InitializedTime,
			LastAddNodeTime: s.chainState.LastAddNodeTime,
		}
	} else {
		return &ChainState{
			Nodes: nodes,
		}
	}

}

// RPC Method
// Broadcast: "init all clients" with per-client NodeID assignment.
func (s *SetupService) BroadcastInitializeChainNode(
	chainID string,
	nodeArray string,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.chainState.Initialized() {
		return ReturnError(errors.New("chain already initialized"))
	}

	newChainState := NewChainState(chainID, nodeArray)

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {
		nodecmds[nodeID] = &nodeCommand{
			cmd: common.CmdInitializeChainNode,
			payload: common.SetupInitializeChainNode{
				NodeID:     nodeID,
				GenesisDoc: *newChainState.GenesisDoc,
				NodeArray:  newChainState.Peers,
			},
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	//succeed
	//save chain node information, genesisDoc
	if err := newChainState.Save(chainStateFile); err != nil {
		return fmt.Errorf("save chain state file: %v, err: %v", chainStateFile, err)
	} else {
		s.chainState = newChainState
	}

	return nil
}

// RPC Method
// Broadcase: add one node and update others nodes.
func (s *SetupService) BroadcastAddChainNode(peerInfo lib.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("please initialize chain node first"))
	}

	newChainState := s.chainState.CloneAddNode(peerInfo)

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {

		var (
			cmd           common.CmdType
			payloadStruct interface{}
		)

		if nodeID == peerInfo.NodeId {
			// send CmdInitializeChainNode to new node
			cmd = common.CmdInitializeChainNode
			payloadStruct = common.SetupInitializeChainNode{
				NodeID:     peerInfo.NodeId,
				NodeArray:  newChainState.Peers,
				GenesisDoc: *s.chainState.GenesisDoc,
			}
		} else {
			// send CmdUpdateChainCfg to other nodes
			cmd = common.CmdUpdateChainCfg
			payloadStruct = common.SetupUpdateChainCfg{
				NodeArray: newChainState.Peers,
			}
		}

		nodecmds[nodeID] = &nodeCommand{
			cmd:     cmd,
			payload: payloadStruct,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		log.Println("send node command failure:", err)
		return err
	}

	//save chain node information, genesisDoc
	if err := newChainState.Save(chainStateFile); err != nil {
		return fmt.Errorf("save chain state file: %v, err: %v", chainStateFile, err)
	} else {
		s.chainState = newChainState
	}

	return nil
}

// RPC Method
// send start,stop,restart command to specified node
func (s *SetupService) ControlNode(ctlNodes []common.SetupControlNode) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("please initialize chain node first"))
	}

	nodecmds := make(map[string]*nodeCommand, len(ctlNodes))

	for _, node := range ctlNodes {
		nodecmds[node.NodeID] = &nodeCommand{
			cmd:     common.CmdControlNode,
			payload: node,
		}

		log.Printf("[ControlNode] nodeID: %v, action: %v\n", node.NodeID, node.Action)
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	return nil
}

// RPC Method
// reset chain state to first initialized state
func (s *SetupService) BroadcastResetChainNode(resetChain common.SetupResetChain) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("chain has not initialized"))
	}

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {
		nodecmds[nodeID] = &nodeCommand{
			cmd:     common.CmdResetChain,
			payload: resetChain,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	//clean event log up
	if err := s.cmdEvent.Reset(); err != nil {
		return fmt.Errorf("failed to reset cmd event: %v", err)
	}

	if err := s.chainState.Reset(chainStateFile); err != nil {
		return fmt.Errorf("failed to reset chain state: %v", err)
	}
	return nil
}

// Method
// clean all block data and height. might experiment.
func (s *SetupService) BroadcastPurgeData() error {
	panic("implement me")
}

// arg
// key: nodeID
// value: {cmd, payload}
func (s *SetupService) sendNodeCommand(cmds map[string]*nodeCommand) error {
	var syncCmds []string

	for nodeID, cmd := range cmds {

		cmdID, err := newCommandID()
		if err != nil {
			return ReturnError(err)
		}

		syncCmds = append(syncCmds, cmdID)

		payload, _ := cmtjson.Marshal(cmd.payload)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       cmds[nodeID].cmd,
			NodeID:    nodeID,
			Payload:   payload,
			IssuedAt:  time.Now().UnixNano(),
		}

		//put command record into database
		_ = s.cmdEvent.Update(commandlog.CommandResult{
			CommandID: cmdID,
			Cmd:       env.Cmd,
			NodeID:    nodeID,
			Status:    commandlog.CmdRunning,
			Message:   string(payload),
			StartedAt: time.Now().UnixNano(),
		}, nil)

		requestNodeFn := func() error {
			clientSub := s.getNotifierByNodeID(nodeID)
			if clientSub == nil {
				return ReturnError(errors.New("client not initialized"))
			}

			//notify all client to initialize node
			if err := clientSub.Notifier.Notify(clientSub.SubID, env); err != nil {
				return fmt.Errorf("Failed to notify node: %v, err %v\n", nodeID, err)
			}
			return nil
		}

		_ = s.cmdManager.ActivateCommand(
			context.Background(),
			command_manager.ActivateOptions{
				ID:      cmdID,
				Timeout: 30 * time.Second,
				StartFn: func(ctx context.Context, id string) error {
					return requestNodeFn()
				},
			})

	}

	//wait all node completing initialization task
	ch := s.cmdManager.SyncMultipleCommands(syncCmds...)

	resErr := &MultiError{
		Errors: make(map[string]error),
	}

	for result := range ch {
		if result.Err != nil {
			resErr.Errors[result.ID] = result.Err
		}
	}

	if len(resErr.Errors) == 0 {
		return nil
	}

	return resErr
}

// internal method
func (s *SetupService) getNotifierByNodeID(nodeID string) *ClientSub {

	clientSub, ok := s.subs[nodeID]
	if !ok {
		return nil
	}

	return clientSub
}

// RPC Method
// complete command that sent to node
func (s *SetupService) ReportCommandResult(res commandlog.CommandResult) error {

	log.Printf("[CommandResult] cmd=%s id=%s node=%s status=%s msg=%q err=%q\n",
		res.Cmd, res.CommandID, res.NodeID, res.Status, res.Message, res.Error)

	if reported := s.stream.ReportState(res); reported {
		//set reported state for cmdEvent
	}

	var err error = nil
	if res.Error != "" {
		err = errors.New(res.Error)
	}

	_ = s.cmdManager.CompleteCommand(res.CommandID, err)

	_, err = s.cmdEvent.Complete(res.CommandID, res.Status == commandlog.CmdSuccess, res.Message, res.Error, res.Output)

	return err
}
