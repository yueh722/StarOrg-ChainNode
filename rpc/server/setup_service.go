package server

import (
	"chain/lib"
	"chain/lib/logdb"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/utils"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pingcap/errors"
)

type SetupResult struct {
	Error error `json:"error"`
}

type nodeCommand struct {
	cmd     common.CmdType
	payload interface{}
}

// SetupService is designed for nodeproxy getting setup command dispatched from NodeMaster.
// Request from: NodeProxy
type SetupService struct {
	mu         sync.Mutex
	subs       map[string]*ClientSub
	stream     *StreamService
	cmdEvent   commandlog.CommandEventRepo
	chainState *ChainStateInternal
}

type ClientSub struct {
	Notifier   *rpc.Notifier
	NodeID     string
	StatusNode common.StatusNode
	SubID      rpc.ID
}

var chainStateFile = filepath.Join(utils.GetAppDataPath("setup"), "chain.json")

func NewSetupService(store logdb.CommandLogStore, stream *StreamService) *SetupService {

	cmdEvent, err := commandlog.NewCommandRepo(store)
	if err != nil {
		log.Fatal(err)
	}

	return &SetupService{
		subs:       make(map[string]*ClientSub),
		cmdEvent:   cmdEvent,
		stream:     stream,
		chainState: LoadChainState(chainStateFile),
	}
}

func (s *SetupService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("chain", s); err != nil {
		return fmt.Errorf("Failed to register chain service:", err)
	}

	return nil
}

func (s *SetupService) GetNamesapceString() string {
	return "chain"
}

func (s *SetupService) GetSubscriptionName() string {
	return "subscribeCommand"
}

func ReturnError(err error) error {
	return err
}

// Method
// Fetch: "retrive chain state and node state"
func (s *SetupService) GetChainState() ChainState {

	var nodes []common.StatusNode

	for _, peer := range s.chainState.Peers {

		node := lib.PeerInfoToNodeInfo(&peer)

		nodes = append(nodes, common.StatusNode{
			NodeInfo:    *node,
			Healthy:     false,
			LastCheck:   time.Now().Unix(),
			StartedTime: "",
		})
	}

	return ChainState{
		Nodes:           nodes,
		ChainID:         s.chainState.GenesisDoc.ChainID,
		InitializedTime: s.chainState.InitializedTime,
		LastAddNodeTime: s.chainState.LastAddNodeTime,
	}
}

// Method
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
		log.Panicln("save chain state file:", chainStateFile, "err:", err)
	} else {
		s.chainState = newChainState
	}

	return nil
}

// Method
func (s *SetupService) BroadcastAddChainNode(peerInfo lib.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("chain has not initialized"))
	}

	//newChainState := s.chainState

	peers := append(s.chainState.Peers, peerInfo)

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
				NodeArray:  peers,
				GenesisDoc: *s.chainState.GenesisDoc,
			}
		} else {
			// send CmdUpdateChainCfg to other nodes
			cmd = common.CmdUpdateChainCfg
			payloadStruct = common.SetupUpdateChainCfg{
				NodeArray: peers,
			}
		}

		nodecmds[nodeID] = &nodeCommand{
			cmd:     cmd,
			payload: payloadStruct,
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	//save chain node information, genesisDoc
	newState := s.chainState.CloneAddNode(peerInfo)
	if err := newState.Save(chainStateFile); err != nil {
		log.Panicln("save chain state file:", chainStateFile, "err:", err)
	} else {
		s.chainState = newState
	}

	return nil
}

// Method
// send start,stop,restart command to specified node
func (s *SetupService) ControlNodes(ctlNodes ...common.SetupControlNode) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("chain has not initialized"))
	}

	for _, node := range ctlNodes {

		cmdID, err := newCommandID()
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(node)

		log.Printf("[ControlNodes] nodeID: %v, action: %v\n", node.NodeID, node.Action)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       common.CmdControlNode,
			NodeID:    node.NodeID,
			Payload:   payload,
			IssuedAt:  time.Now().Unix(),
		}

		clientSub := s.GetNotifierByNodeID(node.NodeID)
		if clientSub != nil {
			if err := clientSub.Notifier.Notify(clientSub.SubID, env); err != nil {
				log.Printf("[ControlNodes] Failed to notify node: %v name: %v, err %v\n",
					node.NodeID,
					"",
					err)
				return err
			}
		}
	}

	return nil
}

// Method
// reset chain state to first initialized state
func (s *SetupService) BroadcastResetChainNode() error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("chain has not initialized"))
	}

	nodecmds := make(map[string]*nodeCommand, len(s.subs))

	for nodeID, _ := range s.subs {
		nodecmds[nodeID] = &nodeCommand{
			cmd:     common.CmdResetChain,
			payload: common.SetupResetChain{},
		}
	}

	if err := s.sendNodeCommand(nodecmds); err != nil {
		return err
	}

	s.chainState = &ChainStateInternal{}

	return nil
}

// Method
// clean all block data and height. might experiment.
func (s *SetupService) BroadcastPurgeData() error {
	return nil
}

// arg
// key: nodeID
// value: {cmd, payload}
func (s *SetupService) sendNodeCommand(cmds map[string]*nodeCommand) error {

	var (
		succeedCount int32 = 0
		nodeErrs           = make(map[string]error)
		wg           sync.WaitGroup
	)

	for nodeID, cmd := range cmds {

		cmdID, err := newCommandID()
		if err != nil {
			return ReturnError(err)
		}

		payload, _ := json.Marshal(cmd.payload)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       cmds[nodeID].cmd,
			NodeID:    nodeID,
			Payload:   payload,
			IssuedAt:  time.Now().Unix(),
		}

		clientSub := s.GetNotifierByNodeID(nodeID)
		if clientSub == nil {
			return ReturnError(errors.New("client not initialized"))
		}

		//notify all client to initialize node
		if err := clientSub.Notifier.Notify(clientSub.SubID, env); err != nil {
			return ReturnError(err)
		}

		//put command record into database
		_ = s.cmdEvent.Update(commandlog.CommandResult{
			CommandID: cmdID,
			Cmd:       env.Cmd,
			NodeID:    nodeID,
			Status:    commandlog.CmdRunning,
			Message:   string(payload),
			StartedAt: time.Now().Unix(),
		}, func(err error) {
			if err == nil {
				atomic.AddInt32(&succeedCount, 1)
			}

			nodeErrs[nodeID] = err
			wg.Add(1)
		})
	}

	//wait all node complete initialization task
	wg.Wait()

	var err = errors.New("")

	if int(succeedCount) == len(s.subs) {
		//all task succeeded
		return nil
	}

	maputil.ForEach(nodeErrs, func(nodeID string, nodeErr error) {
		err = fmt.Errorf("%v\nnodeID: %v, err: %v", err, nodeID, nodeErr.Error())
	})

	return err
}

func (s *SetupService) GetNotifierByNodeID(nodeID string) *ClientSub {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientSub, ok := s.subs[nodeID]
	if !ok {
		return nil
	}

	return clientSub
}

// Method
// complete command that sent to node before
func (s *SetupService) ReportCommandResult(res commandlog.CommandResult) error {

	log.Printf("[CommandResult] cmd=%s id=%s node=%s status=%s progress=%.1f msg=%q err=%q\n",
		res.Cmd, res.CommandID, res.NodeID, res.Status, res.Message, res.Error)

	if reported := s.stream.ReportState(res); reported {
		//set reported state for cmdEvent
	}

	_, err := s.cmdEvent.Complete(res.CommandID, res.Status == commandlog.CmdSuccess, res.Message, res.Error, res.Output)

	return err
}
