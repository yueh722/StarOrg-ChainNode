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
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pingcap/errors"
)

type SetupResult struct {
	Error error `json:"error"`
}

// SetupService is designed for nodeproxy getting setup command dispatched from NodeMaster.
// Request from: NodeProxy
type SetupService struct {
	mu         sync.Mutex
	subs       map[rpc.ID]*ClientSub
	stream     *StreamService
	cmdEvent   commandlog.CommandEventRepo
	chainState *ChainStateInternal
}

type ClientSub struct {
	Notifier *rpc.Notifier
	NodeID   string
	NodeInfo lib.NodeInfo
}

var chainStateFile = filepath.Join(utils.GetAppDataPath("setup"), "chain.json")

func NewSetupService(store logdb.CommandLogStore, stream *StreamService) *SetupService {

	cmdEvent, err := commandlog.NewCommandRepo(store)
	if err != nil {
		log.Fatal(err)
	}

	return &SetupService{
		subs:       make(map[rpc.ID]*ClientSub),
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
			NodeId:         node.NodeID,
			NodeName:       node.NodeName,
			NodeAddress:    node.NodeAddress,
			Validator:      node.Validator,
			ProxyApp:       node.ProxyApp,
			RPCAddress:     node.RPCAddr,
			WebServiceAddr: node.WebServiceAddr,
			Healthy:        false,
			LastCheck:      time.Now().Unix(),
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

	activatedStatus := map[string]error{}

	for subID, clientSub := range s.subs {
		nodeID := clientSub.NodeID

		activatedStatus[nodeID] = nil

		cmdID, err := newCommandID()
		if err != nil {
			return ReturnError(err)
		}

		payloadStruct := common.SetupInitializeChainNode{
			NodeID:     nodeID,
			GenesisDoc: *newChainState.GenesisDoc,
			NodeArray:  newChainState.Peers,
		}
		payload, _ := json.Marshal(payloadStruct)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       common.CmdInitializeChainNode,
			NodeID:    nodeID,
			Payload:   payload,
			IssuedAt:  time.Now().Unix(),
		}

		//notify all client to initialize node
		if err := clientSub.Notifier.Notify(subID, env); err != nil {
			return ReturnError(err)
		}

		//put command record into database
		err = s.cmdEvent.Update(commandlog.CommandResult{
			CommandID: cmdID,
			Cmd:       env.Cmd,
			NodeID:    nodeID,
			Status:    commandlog.CmdRunning,
			Message:   string(payload),
			StartedAt: time.Now().Unix(),
		})
		if err != nil {
			return ReturnError(err)
		}

		_, activatedStatus[nodeID] = s.cmdEvent.WaitCompleted(cmdID)
	}

	var err = errors.New("")

	completed := maputil.FilterByValues(activatedStatus, []error{nil})
	if len(completed) == 0 {
		//all completed
		//save chain node information, genesisDoc
		if err = newChainState.Save(chainStateFile); err != nil {
			log.Println("save chain state file:", chainStateFile, "err:", err)
		}

		s.chainState = newChainState
	}

	maputil.ForEach(activatedStatus, func(nodeID string, opErr error) {
		err = fmt.Errorf("%v\nnodeID: %v, err: %v", err, nodeID, opErr)
	})

	return ReturnError(err)
}

// Method
func (s *SetupService) BroadcastAddChainNode(peerInfo lib.Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.chainState.Initialized() {
		return ReturnError(errors.New("chain has not initialized"))
	}

	cmdID, err := newCommandID()
	if err != nil {
		return err
	}

	activatedStatus := map[string]error{}

	peers := append(s.chainState.Peers, peerInfo)

	notifiers := s.GetNotifierByNodeID(peerInfo.NodeId)

	for subID, clientSub := range notifiers {
		var (
			payloadStruct any
			cmd           common.CmdType
		)

		if clientSub.NodeID == peerInfo.NodeId {
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

		payload, _ := json.Marshal(payloadStruct)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       cmd,
			NodeID:    peerInfo.NodeId,
			Payload:   payload,
			IssuedAt:  time.Now().Unix(),
		}

		if err = clientSub.Notifier.Notify(subID, env); err == nil {
			break
		}

		//put command record into database
		err = s.cmdEvent.Update(commandlog.CommandResult{
			CommandID: cmdID,
			Cmd:       env.Cmd,
			NodeID:    peerInfo.NodeId,
			Status:    commandlog.CmdRunning,
			Message:   string(payload),
			StartedAt: time.Now().Unix(),
		})
		if err != nil {
			return ReturnError(err)
		}

		_, activatedStatus[peerInfo.NodeId] = s.cmdEvent.WaitCompleted(cmdID)
	}

	if len(activatedStatus) == 0 {
		// all down: update chainstate
		err = s.chainState.AddNode(peerInfo).Save(chainStateFile)
	}

	return err
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

		notifiers := s.GetNotifierByNodeID(node.NodeID)

		maputil.ForEach(notifiers, func(subID rpc.ID, clientSub *ClientSub) {
			if err := clientSub.Notifier.Notify(subID, env); err != nil {
				log.Printf("[ControlNodes] Failed to notify node: %v name: %v, err %v\n",
					node.NodeID,
					"",
					err)
			}
		})
	}

	return nil
}

func (s *SetupService) GetNotifierByNodeID(nodeID string) map[rpc.ID]*ClientSub {

	notifiers := maputil.Filter(s.subs, func(_ rpc.ID, clientSub *ClientSub) bool {
		return clientSub.NodeID == nodeID
	})

	return notifiers
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
