package server

import (
	"chain/lib"
	"chain/lib/cmdlogstore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/rpc/common/callclient"
	"chain/rpc/common/notification"
	"chain/rpc/server/command_manager"
	"chain/utils"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"path/filepath"
	"sync"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/cometbft/cometbft/v2/types"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pingcap/errors"
)

type SetupResult struct {
	Error error `json:"error"`
}

type nodeCommand struct {
	cmd     common.CmdType
	payload interface{}
	ctx     context.Context
}

// SetupService is designed for nodeagent getting setup command dispatched from NodeMaster.
// Request from: NodeProxy
type SetupService struct {
	mu            sync.Mutex
	subs          map[string]*ClientSub
	stream        *StreamService
	cmdEvent      commandlog.CommandEventRepo
	chainState    *ChainStateInternal
	cmdManager    *command_manager.CmdManager
	NodeNameCache map[string]string
	notifierMgr   *notification.Manager
}

type ClientSub struct {
	Notifier   *rpc.Notifier
	Client     *rpc.Client
	NodeID     string
	DeviceID   string
	StatusNode common.StatusNode
	SubID      rpc.ID
}

var chainStateFile = filepath.Join(utils.GetAppDataPath("SetupService"), "setup.json")

func init() {
	utils.TryCreateDir(filepath.Dir(chainStateFile))
}

func NewSetupService(store cmdlogstore.CommandLogStore, stream *StreamService, notifierMgr *notification.Manager) *SetupService {

	cmdEvent, err := commandlog.NewCommandRepo(store)
	if err != nil {
		log.Panic(err)
	}

	chainstate, err := LoadChainState(chainStateFile)
	if err != nil {
		chainstate = &ChainStateInternal{}
	}

	return &SetupService{
		subs:          make(map[string]*ClientSub),
		cmdEvent:      cmdEvent,
		stream:        stream,
		chainState:    chainstate,
		cmdManager:    command_manager.NewCmdManager(),
		NodeNameCache: make(map[string]string),
		notifierMgr:   notifierMgr,
	}
}

func (s *SetupService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(common.SetupNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register chain service: %v", err)
	}

	return nil
}

// RPC Method
// Fetch: "retrive chain state and node state"
func (s *SetupService) GetChainState() *ChainState {
	var nodeMap = map[string]*common.StatusNode{}

	s.mu.Lock()
	defer s.mu.Unlock()

	//convert initialized peers to statusNodes first from chainStateInternal
	nodeMap = convertor.ToMap(s.chainState.Peers, func(peerInfo lib.Peer) (string, *common.StatusNode) {
		return peerInfo.NodeId, &common.StatusNode{
			NodeInfo:    *lib.PeerInfoToNodeInfo(&peerInfo),
			Initialized: true,
		}
	})

	//init other members from subscribers
	for nodeID, clientSub := range s.subs {
		node := &clientSub.StatusNode
		if node.Initialized {
			node.Healthy = s.IsNodeRpcAlive(nodeID)
			node.LastCheck = utils.NowString()
		}

		//update statusNode if node is a command subscriber
		nodeMap[nodeID] = node
	}

	nodes := convertor.MapToSlice(nodeMap, func(_ string, statusNode *common.StatusNode) common.StatusNode {
		return *statusNode
	})

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
// fetch: retrive history of setup command
func (s *SetupService) GetHistoryOfCommand(nodeID string) ([]commandlog.CommandResult, error) {

	nodeIDs := []string{}
	if nodeID != "" {
		nodeIDs = append(nodeIDs, nodeID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodeIDs) == 0 {
		nodeIDs = maputil.Keys(s.NodeNameCache)
	}

	results := []commandlog.CommandResult{}
	for _, id := range nodeIDs {
		result, err := s.cmdEvent.ListCommandByNode(id)
		if err != nil {
			return nil, err
		}

		results = append(results, result...)
	}

	return results, nil
}

// RPC Method
// Broadcast: "init all node" with per-node NodeID assignment.
func (s *SetupService) BroadcastInitializeChainNode(
	chainID string,
	nodeArray []lib.ArgNodeInfo,
) (*common.RpcResponse, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.chainState.Initialized() {
		return common.MakeRpcErrorf("chain already initialized")
	}

	if len(nodeArray) == 0 {
		return common.MakeRpcErrorf("argument node array is empty")
	}

	peers := lib.ArgNodeArrayToPeers(nodeArray)

	if err := lib.ValidNodeArray(peers); err != nil {
		return common.MakeRpcError(err)
	}

	newChainState := NewChainState(chainID, peers)

	nodecmds := map[string]*nodeCommand{}

	for nodeID, _ := range s.subs {
		peer := peers.GetPeerByNodeId(nodeID)
		if peer.NodeId != nodeID {
			log.Infof("Peer is not in the initialization list, Ignore sending init command with peer %v to node",
				s.FormatName(nodeID))
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

		log.Info("build init command with node:", s.FormatName(nodeID))
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return common.MakeRpcError(err)
	}

	// update initialization status of nodes that had finished.
	s.PeerInitializationCompleted(maputil.Keys(nodecmds))

	//succeed
	//save chain node information, genesisDoc
	if err := newChainState.Save(chainStateFile); err != nil {
		return common.MakeRpcErrorf("failed to save chain state file: %v, err: %v", chainStateFile, err)
	} else {
		s.chainState = newChainState
	}

	log.Info("initializeChainNode complete", chainID)

	return common.MakeRpcSucceed()
}

// RPC Method
// Broadcase: add one node and update others nodes.
func (s *SetupService) BroadcastAddChainNode(node lib.ArgNodeInfo) (*common.RpcResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return common.MakeRpcErrorf("please initialize chain first")
	}

	if s.chainState.IsNodeInitialized(node.NodeId) {
		return common.MakeRpcErrorf("this node had already initialized")
	}

	if !s.HasActivatedValidator() {
		return common.MakeRpcErrorf("activated validator nodes not found, please run all nodes first before add a new node")
	}

	peerInfo := node.ToPeer()

	peers := lib.PeerList{node.ToPeer()}
	if err := lib.ValidNodeArray(peers); err != nil {
		return common.MakeRpcError(err)
	}

	peerInfo = peers[0]
	newChainState := s.chainState.CloneAddNode(peerInfo)

	nodecmds := make(map[string]*nodeCommand)

	log.Info("prepare add node:", s.FormatName(peerInfo.NodeId))
	log.Info("peers already initialized: ", slice.Map(s.chainState.Peers, func(_ int, peer lib.Peer) string {
		return fmt.Sprintf("%v : %v\n",
			s.FormatName(peer.NodeId),
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

			log.Info("build init command with node", s.FormatName(peerInfo.NodeId))

		} else {

			if !clientStub.StatusNode.Initialized {
				log.Info("this node has not initialized, Ignore sending update command to it, ",
					s.FormatName(nodeID))
				continue
			}

			// send CmdUpdateChainCfg to other nodes that had initialized before
			cmd = common.CmdUpdateChainCfg
			payloadStruct = common.SetupUpdateChainCfg{
				NodeArray: newChainState.Peers,
			}

			log.Info("build update chain config command with node", s.FormatName(nodeID))

		}

		nodecmds[nodeID] = &nodeCommand{
			cmd:     cmd,
			payload: payloadStruct,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		log.Info("send node command failure:", err)
		return common.MakeRpcError(err)
	}

	s.NodeNameCache[peerInfo.NodeId] = peerInfo.NodeName

	// update initialization status of new node.
	s.PeerInitializationCompleted([]string{peerInfo.NodeId})

	//save chain node information, genesisDoc
	if err := newChainState.Save(chainStateFile); err != nil {
		return common.MakeRpcErrorf("failed to save chain state file: %v, err: %v", chainStateFile, err)
	} else {
		s.chainState = newChainState
	}

	//submit adding validator for current peer if it's set to be a validator
	if peerInfo.IsValidator {
		if err := s.BroadcaseAddValidator(peerInfo); err != nil {
			log.Info("broadcast add validator failure:", err)
			return common.MakeRpcError(err)
		}
	}

	return common.MakeRpcSucceed()
}

func (s *SetupService) isOriginalValidator(nodeName string) bool {
	_, isValidator := slice.FindBy(s.chainState.GenesisDoc.Validators, func(_ int, v types.GenesisValidator) bool {
		return strings.ToLower(v.Name) == strings.ToLower(nodeName)
	})
	return isValidator
}

func (s *SetupService) BroadcaseAddValidator(peerInfo lib.Peer) error {
	if !s.chainState.Initialized() {
		return errors.New("please initialize chain node first before broadcast add validator")
	}

	var totalErr = lib.NewMultiError()

	for nodeID, clientStub := range s.subs {

		if !s.isOriginalValidator(clientStub.StatusNode.NodeName) {
			//no need to send the command to a non-validator node.
			continue
		}

		if !s.IsNodeRpcAlive(nodeID) {
			//no need to send the command to an offline node.
			log.Infof("ignore the node: %v for adding validator:", s.FormatName(peerInfo.NodeId))
			totalErr.Errors[nodeID] = fmt.Errorf("RPC of the node is not alive.")
			continue
		}

		v := peerInfo.BuildValidator()
		nodecmds := make(map[string]*nodeCommand)

		nodecmds[nodeID] = &nodeCommand{
			cmd: common.CmdAddValidator,
			payload: common.SetupAddValidator{
				AddValidatorReq: lib.AddValidatorReq{
					PubType: v.PubKey.Type(),
					PubKey:  base64.StdEncoding.EncodeToString(v.PubKey.Bytes()),
					Power:   v.Power / 2,
					Name:    v.Name,
				},
			},
		}

		if err := s.sendNodeCommand(nodecmds); err != nil {
			log.Info("send node command failure:", err)
			totalErr.Errors["err"] = err
			continue
		}

		totalErr.Clear()
		break
	}

	return totalErr.ErrorOrNil()
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
func (s *SetupService) ControlNode(ctlNodes []common.SetupControlNode) (*common.RpcResponse, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return common.MakeRpcErrorf("please initialize chain node first")
	}

	nodecmds := make(map[string]*nodeCommand)

	for _, ctlNode := range ctlNodes {
		if !s.chainState.IsNodeInitialized(ctlNode.NodeID) {
			log.Infof("the node %v has not initialized, Ignore sending control command to it",
				s.FormatName(ctlNode.NodeID))
			continue
		}

		nodecmds[ctlNode.NodeID] = &nodeCommand{
			cmd:     common.CmdControlNode,
			payload: ctlNode,
		}

		log.Infof("[ControlNode] node: %v, action: %v", s.FormatName(ctlNode.NodeID), ctlNode.Action)
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return common.MakeRpcError(err)
	}

	return common.MakeRpcSucceed()
}

// RPC Method
// reset chain state to uninitialized state, remove all data of node and app
func (s *SetupService) BroadcastPurgeChainNode(purgeChain common.SetupPurgeChain) (*common.RpcResponse, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return common.MakeRpcErrorf("chain has not been initialized")
	}

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {
		nodecmds[nodeID] = &nodeCommand{
			cmd:     common.CmdPurgeChain,
			payload: purgeChain,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return common.MakeRpcError(err)
	}

	//clean event log up
	if err := s.cmdEvent.Reset(); err != nil {
		return common.MakeRpcErrorf("failed to reset cmd event: %v", err)
	}

	if err := s.chainState.Reset(chainStateFile); err != nil {
		return common.MakeRpcErrorf("failed to reset chain state file: %v, err: %v", chainStateFile, err)
	}

	s.ResetNodeStatus()

	return common.MakeRpcSucceed()
}

// arg
// key: nodeID
// value: {cmd, payload}
func (s *SetupService) sendNodeCommand(cmds map[string]*nodeCommand) error {
	var syncCmds []string

	for nodeID, cmd := range cmds {

		cmdID := utils.UUID()

		syncCmds = append(syncCmds, cmdID)

		payload, _ := cmtjson.Marshal(cmd.payload)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       cmds[nodeID].cmd,
			NodeID:    nodeID,
			Payload:   payload,
			IssuedAt:  utils.NowString(),
		}

		//put command record into database
		_ = s.cmdEvent.Update(commandlog.CommandResult{
			CommandID: cmdID,
			Cmd:       env.Cmd,
			NodeID:    nodeID,
			Status:    commandlog.CmdRunning,
			Message:   string(payload),
			StartedAt: utils.NowString(),
		}, nil)

		requestNodeFn := func() error {
			clientSub := s.getNotifierByNodeID(nodeID)
			if clientSub == nil {
				return (errors.New("client not found in subscription list"))
			}

			log.Infof("sending command @%v to node %v", env.Cmd, s.FormatName(nodeID))

			//notify all client to initialize node
			if err := clientSub.Notifier.Notify(clientSub.SubID, env); err != nil {
				return fmt.Errorf("Failed to notify node: %v, err %v", s.FormatName(nodeID), err)
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

	resErr := lib.NewMultiError()

	for result := range ch {
		if result.Err != nil {
			resErr.Errors[result.ID] = result.Err
		}
	}

	return resErr.ErrorOrNil()
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
func (s *SetupService) ReportCommandResult(ctx context.Context, res commandlog.CommandResult) error {

	log.Printf("[CommandResult] cmd: %s, node: %v, status: %s, err: %v",
		res.Cmd, s.FormatName(res.NodeID), res.Status, res.Error)

	//notify the stream's subscribers that one command is completed.
	if reported := s.stream.ReportState(&res); reported {

	}

	var err error = nil
	if res.Error != "" {
		err = errors.New(res.Error)
	}

	// signal command to let the reader get the result of the command.
	_ = s.cmdManager.CompleteCommand(res.CommandID, err)

	// set the status of command to finished, which will be saved to local db.
	_, err = s.cmdEvent.Complete(res.CommandID, res.Status == commandlog.CmdSuccess, res.Message, res.Error)

	return nil
}

func (s *SetupService) FormatName(nodeID string) string {
	if name, ok := s.NodeNameCache[nodeID]; ok {
		return fmt.Sprintf("%s | %s", name, nodeID)
	}
	return nodeID
}

func (s *SetupService) IsNodeRpcAlive(nodeID string) bool {
	clientSub := s.getNotifierByNodeID(nodeID)
	if clientSub == nil {
		log.Info("clientSub not found, node: %v", s.FormatName(nodeID))
		return false
	}

	return callclient.IsNodeRpcAlive(clientSub.Client)
}

func (s *SetupService) HasActivatedValidator() bool {

	for nodeID, clientStub := range s.subs {

		if !s.isOriginalValidator(clientStub.StatusNode.NodeName) {
			//no need to send this kind of command to a non-validator node.
			continue
		}

		if s.IsNodeRpcAlive(nodeID) {
			log.Info("validator node is online, node: ", s.FormatName(nodeID))
			return true
		}

		log.Info("validator node is offline, node: ", s.FormatName(nodeID))
	}

	return false
}
