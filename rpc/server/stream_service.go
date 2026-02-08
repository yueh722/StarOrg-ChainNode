package server

import (
	"chain/lib"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"chain/rpc/common/notification"
	"chain/utils"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type customerSub struct {
	notifier *rpc.Notifier
	subID    rpc.ID
}

// StreamService is used to provide live stream data about chain network for webserver.
type StreamService struct {
	subs             map[rpc.ID]customerSub
	mut              sync.Mutex
	initializedNodes map[string]bool
}

func NewStreamService() *StreamService {
	return &StreamService{
		subs: make(map[rpc.ID]customerSub),
		mut:  sync.Mutex{},
	}
}

func (s *StreamService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(common.StreamNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register stream service: %v", err)
	}

	return nil
}

func (s *StreamService) GetCustomerCount() int {

	s.mut.Lock()
	defer s.mut.Unlock()

	return len(s.subs)
}

// SubscribeErrors：JSON-RPC subscription
// Subscriber call method: "stream_subscribeStream"
// subscirbe stream to receive live status about node
func (s *StreamService) SubscribeStream(ctx context.Context, arg StreamSubscribeArg) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()

	s.mut.Lock()
	defer s.mut.Unlock()

	s.subs[sub.ID] = customerSub{
		notifier: notifier,
		subID:    sub.ID,
	}

	go func() {
		<-sub.Err()
		s.mut.Lock()
		delete(s.subs, sub.ID)
		s.mut.Unlock()
	}()

	return sub, nil
}

func (s *StreamService) OnNotify(msg *notification.Message) {

	switch msg.Type {
	case notification.MsgTypeOffline, notification.MsgTypeOnline:
		s.ReportState(&common.StatusAgent{
			Catalog:  "agent",
			NodeID:   msg.Payload.(string),
			Status:   msg.Type,
			IssuedAt: utils.NowString(),
		})
	case notification.MsgTypeInitialized, notification.MsgTypeUnInitialized:
		s.mut.Lock()
		defer s.mut.Unlock()
		s.initializedNodes[msg.Payload.(string)] = (msg.Type == notification.MsgTypeInitialized)
	}
}

func (s *StreamService) ReportState(state interface{}) bool {
	// ReportState receives following kind of data and will transmit these data to subscribers.

	// ReportState receives following kind of data and will transmit these data to subscribers.
	// *common.StatusNode
	// *common.StatusStorage
	// *commandlog.CommandResult
	// *lib.StatusTransaction
	// *common.StatusCometbftProcess
	// *common.StatusAgent

	var rpcState struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	/*
		var dropStatus bool
		needToDropStatusFn := func(nodeID string) bool {
			s.mut.Lock()
			defer s.mut.Unlock()
			return !s.initializedNodes[nodeID]
		}
	*/

	rpcState.Data = state

	switch v := state.(type) {
	case *common.StatusNode:
		rpcState.Type = "node"
	case *common.StatusStorage:
		rpcState.Type = "storage"
	case *commandlog.CommandResult:
		rpcState.Type = "setup"
	case *lib.StatusTransaction:
		rpcState.Type = "tx"
	case *queuestore.PersistedEvent:
		rpcState.Type = "log"
	case *common.StatusCometbftProcess:
		rpcState.Type = "process"
	case *common.StatusAgent:
		rpcState.Type = "agent"
	default:
		log.Info("ReportState unknown state", v)
		return true
	}

	return s.reportStateInternal(rpcState)
}

func (s *StreamService) reportStateInternal(state interface{}) bool {
	reported := true
	var err error

	err = s.notifyClient(state)
	if err != nil {
		log.Error("notify stream data error", err)
		reported = false
	}

	return reported
}

func (s *StreamService) notifyClient(data any) error {
	var err error

	s.mut.Lock()
	defer s.mut.Unlock()

	for _, sub := range s.subs {
		err2 := sub.notifier.Notify(sub.subID, data)
		if err == nil {
			err = err2
		}
	}
	return err
}
