package server

import (
	"chain/lib"
	"chain/lib/logdb"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
)

// SetupService is designed for nodeproxy getting setup command dispatched from NodeMaster.
// Request from: NodeProxy
type SetupService struct {
	mu     sync.Mutex
	subs   map[rpc.ID]*ClientSub
	stream *StreamService
	//
	cmdEvent commandlog.CommandEventRepo
}

type ClientSub struct {
	Notifier *rpc.Notifier
	NodeID   string
	NodeInfo lib.NodeInfo
}

func NewSetupService(store logdb.CommandLogStore, stream *StreamService) *SetupService {

	cmdEvent, err := commandlog.NewCommandRepo(store)
	if err != nil {
		log.Fatal(err)
	}

	return &SetupService{
		subs:     make(map[rpc.ID]*ClientSub),
		cmdEvent: cmdEvent,
		stream:   stream,
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

// Method
// Broadcast: "init all clients" with per-client NodeID assignment.
func (s *SetupService) BroadcastInitializeChainNode(
	chainID string,
	genesisTime int64,
	nodeArray string,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	for subID, clientSub := range s.subs {
		nodeID := clientSub.NodeID

		cmdID, err := newCommandID()
		if err != nil {
			return err
		}

		payloadStruct := common.SetupInitializeChainNode{
			ChainID:     chainID,
			GenesisTime: genesisTime,
			NodeID:      nodeID,
			NodeArray:   nodeArray,
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
			return err
		}

		//save genesisDoc, peerList,

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
			return err
		}
	}

	return nil
}

// Method
func (s *SetupService) BroadcastAddChainNode(peers, genesisDoc, nodeId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmdID, err := newCommandID()
	if err != nil {
		return err
	}

	notifiers := s.GetNotifierByNodeID(nodeId)

	for subID, clientSub := range notifiers {

		payloadStruct := common.SetupAddChainNode{
			NodeID:     nodeId,
			NodeArray:  peers,
			GenesisDoc: genesisDoc,
		}
		payload, _ := json.Marshal(payloadStruct)

		env := common.CommandEnvelope{
			CommandID: cmdID,
			Cmd:       common.CmdAddChainNode,
			NodeID:    nodeId,
			Payload:   payload,
			IssuedAt:  time.Now().Unix(),
		}

		if err = clientSub.Notifier.Notify(subID, env); err == nil {
			break
		}
	}

	return err
}

// Method
// send start,stop,restart command to specified node
func (s *SetupService) ControlNodes(ctlNodes ...common.SetupControlNode) error {

	s.mu.Lock()
	defer s.mu.Unlock()

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
