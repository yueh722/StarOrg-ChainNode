package server

import (
	"Chain/rpc/commandlog"
	"Chain/rpc/common"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
)

type SetupService struct {
	mu   sync.Mutex
	subs map[rpc.ID]*ClientSub

	// optional: store last result per command
	// results map[string]common.CommandResult
}

type ClientSub struct {
	Notifier *rpc.Notifier
	NodeID   string
}

func NewSetupService() *SetupService {
	return &SetupService{
		subs: make(map[rpc.ID]*ClientSub),
	}
}

func (s *SetupService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("chain", s); err != nil {
		return fmt.Errorf("Failed to register chain service:", err)
	}

	return nil
}

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

		if err := clientSub.Notifier.Notify(subID, env); err != nil {
			return err
		}
	}

	return nil
}

func (s *SetupService) ControlNodes(ctlNodes ...common.SetupControlNode) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, node := range ctlNodes {

		cmdID, err := newCommandID()
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(node)

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

			}
		})
	}

	return nil
}

func (s *SetupService) GetNotifierByNodeID(nodeID string) map[rpc.ID]*ClientSub {

	s.mu.Lock()
	defer s.mu.Unlock()

	notifiers := maputil.Filter(s.subs, func(_ rpc.ID, clientSub *ClientSub) bool {
		return clientSub.NodeID == nodeID
	})

	return notifiers
}

func (s *SetupService) ReportCommandResult(res commandlog.CommandResult) error {

	log.Printf("[CommandResult] cmd=%s id=%s node=%s status=%s progress=%.1f msg=%q err=%q",
		res.Cmd, res.CommandID, res.NodeID, res.Status, res.Message, res.Error)

	return nil
}
