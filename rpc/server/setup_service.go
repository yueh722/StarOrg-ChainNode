package server

import (
	"chain/lib"
	"chain/lib/cmdlogstore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/rpc/server/command_manager"
	"chain/utils"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
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
	mu            sync.Mutex
	subs          map[string]*ClientSub
	stream        *StreamService
	cmdEvent      commandlog.CommandEventRepo
	chainState    *ChainStateInternal
	cmdManager    *command_manager.CmdManager
	NodeNameCache map[string]string
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
		log.Panic(err)
	}

	return &SetupService{
		subs:          make(map[string]*ClientSub),
		cmdEvent:      cmdEvent,
		stream:        stream,
		chainState:    LoadChainState(chainStateFile),
		cmdManager:    command_manager.NewCmdManager(),
		NodeNameCache: make(map[string]string),
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
		if node.Initialized {
			node.Healthy = lib.CheckNodeOnline(node.RPCAddr)
			node.LastCheck = utils.NowString()
		}

		nodes = append(nodes, *node)
	}

	slice.SortBy(nodes, func(a, b common.StatusNode) bool {
		return a.NodeInfo.NodeName < b.NodeInfo.NodeName
	})

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
		return (errors.New("chain already initialized"))
	}

	if err := lib.ValidNodeArray(nodeArray); err != nil {
		return err
	}

	newChainState := NewChainState(chainID, nodeArray)

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {
		peer := lib.GetNodeByNodeId(nodeID, nodeArray)
		if peer == nil || !peer.Valid() {
			log.Infof("Peer is not valid, Ignore sending init command with peer %v to node", nodeID)
			continue
		}

		s.NodeNameCache[nodeID] = peer.NodeName

		nodecmds[nodeID] = &nodeCommand{
			cmd: common.CmdInitializeChainNode,
			payload: common.SetupInitializeChainNode{
				NodeID:     nodeID,
				GenesisDoc: *newChainState.GenesisDoc,
				NodeArray:  newChainState.Peers,
			},
		}

		log.Info("build init command with node:", nodeID, peer.NodeName)
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	// update initialization status of nodes that had finished.
	s.PeerInitializationCompleted(maputil.Keys(nodecmds))

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

	if s.chainState.IsNodeInitialized(peerInfo.NodeId) {
		return ReturnError(errors.New("node already initialized"))
	}

	newChainState := s.chainState.CloneAddNode(peerInfo)

	nodecmds := make(map[string]*nodeCommand)

	log.Info("prepare add node:", peerInfo.NodeId, peerInfo.NodeName)
	log.Info("peers already initialized: ", slice.Map(s.chainState.Peers, func(_ int, peer lib.Peer) string {
		return fmt.Sprintf("%v : %v : %v",
			peer.NodeId,
			peer.NodeName,
			s.chainState.IsNodeInitialized(peer.NodeId))
	}))

	for nodeID, clientStub := range s.subs {

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

			log.Info("build init command with node", nodeID, clientStub.StatusNode.NodeName)

		} else {

			if !clientStub.StatusNode.Initialized {
				log.Info("this node has not initialized, Ignore sending update command to it, ", peerInfo.NodeId, peerInfo.NodeName)
				continue
			}

			// send CmdUpdateChainCfg to other nodes that had initialized before
			cmd = common.CmdUpdateChainCfg
			payloadStruct = common.SetupUpdateChainCfg{
				NodeArray: newChainState.Peers,
			}

			log.Info("build update chain config command with node", nodeID, clientStub.StatusNode.NodeName)

		}

		s.NodeNameCache[nodeID] = peerInfo.NodeName

		nodecmds[nodeID] = &nodeCommand{
			cmd:     cmd,
			payload: payloadStruct,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		log.Error("send node command failure:", err)
		return err
	}

	// update initialization status of new node.
	s.PeerInitializationCompleted([]string{peerInfo.NodeId})

	//save chain node information, genesisDoc
	if err := newChainState.Save(chainStateFile); err != nil {
		return fmt.Errorf("save chain state file: %v, err: %v", chainStateFile, err)
	} else {
		s.chainState = newChainState
	}

	return nil
}

func (s *SetupService) PeerInitializationCompleted(nodeIDs []string) {
	for _, nodeID := range nodeIDs {
		if clientStub, ok := s.subs[nodeID]; ok {
			clientStub.StatusNode.Initialized = true
		}
	}
}

func (s *SetupService) ResetNodeStatus() {
	for _, clientSub := range s.subs {
		clientSub.StatusNode.Initialized = false
		lib.ResetNodeInfo(&clientSub.StatusNode.NodeInfo)
	}
}

// RPC Method
// send start,stop,restart command to specified node
func (s *SetupService) ControlNode(ctlNodes []common.SetupControlNode) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("please initialize chain node first"))
	}

	nodecmds := make(map[string]*nodeCommand)

	for _, ctlNode := range ctlNodes {
		if !s.chainState.IsNodeInitialized(ctlNode.NodeID) {
			log.Warnf("the node %v has not initialized, Ignore sending control command to it", ctlNode.NodeID)
			continue
		}

		nodecmds[ctlNode.NodeID] = &nodeCommand{
			cmd:     common.CmdControlNode,
			payload: ctlNode,
		}

		log.Printf("[ControlNode] nodeID: %v, action: %v", ctlNode.NodeID, ctlNode.Action)
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

	s.ResetNodeStatus()

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

			log.Info("sending command to node", nodeID)

			//notify all client to initialize node
			if err := clientSub.Notifier.Notify(clientSub.SubID, env); err != nil {
				return fmt.Errorf("Failed to notify node: %v, err %v", nodeID, err)
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

	log.Printf("[CommandResult] cmd: %s, nodeID: %s, nodeName: %s, status: %s, err: %v",
		res.Cmd, res.NodeID, s.NodeNameCache[res.NodeID], res.Status, res.Error)

	if reported := s.stream.ReportState(res); reported {
		//set reported state for cmdEvent
	}

	var err error = nil
	if res.Error != "" {
		err = errors.New(res.Error)
	}

	// complete command
	_ = s.cmdManager.CompleteCommand(res.CommandID, err)

	// complete log status of command event
	_, err = s.cmdEvent.Complete(res.CommandID, res.Status == commandlog.CmdSuccess, res.Message, res.Error)

	return err
}
