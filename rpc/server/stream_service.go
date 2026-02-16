package server

import (
	"chain/lib"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/commandlog"
	"chain/rpc/common/namespace"
	"chain/rpc/common/notification"
	"chain/rpc/common/setupTypes"
	"chain/utils"
	"context"
	"errors"
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
	if err := server.RegisterName(namespace.StreamNamespace{}.GetNamesapceString(), s); err != nil {
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
		s.ReportState(&setupTypes.StatusAgent{
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
	case *setupTypes.StatusNode:
		rpcState.Type = "node"
	case *setupTypes.StatusStorage:
		rpcState.Type = "storage"
	case *commandlog.CommandResult:
		rpcState.Type = "setup"
	case *lib.StatusTransaction:
		rpcState.Type = "tx"
	case *queuestore.PersistedEvent:
		rpcState.Type = "log"
	case *setupTypes.StatusCometbftProcess:
		rpcState.Type = "process"
	case *setupTypes.StatusAgent:
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
		reported = false
	}

	return reported
}

func (s *StreamService) notifyClient(data any) error {
	var err = lib.NewMultiError()

	s.mut.Lock()
	defer s.mut.Unlock()

	if len(s.subs) == 0 {
		return errors.New("no subscribers registered")
	}

	for _, sub := range s.subs {
		err2 := sub.notifier.Notify(sub.subID, data)
		if err2 != nil {
			log.Info("notify err:", err2)
			err.AddError(err2)
		}
	}
	return err.ErrorOrNil()
}
