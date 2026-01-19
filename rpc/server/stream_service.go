package server

import (
	log "chain/lib/zlog"
	"chain/rpc/common"
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
	subs      map[rpc.ID]customerSub
	mut       sync.Mutex
	namespace common.Namespace
}

func NewStreamService() *StreamService {
	return &StreamService{
		subs:      make(map[rpc.ID]customerSub),
		mut:       sync.Mutex{},
		namespace: common.StreamNamespace{},
	}
}

func (s *StreamService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(s.namespace.GetNamesapceString(), s); err != nil {
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
func (s *StreamService) SubscribeStream(ctx context.Context, args SubscribeArgs) (*rpc.Subscription, error) {
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

type StreamReportCallback func(*StreamService, interface{}) bool

func (s *StreamService) ReportState(state interface{}) bool {
	reported := true
	var err error

	// ReportState receives following kind of data and will transmit these data to subscribers.
	// *common.StatusNode
	// *common.StatusStorage
	// commandlog.CommandResult
	// lib.StatusTransaction

	err = s.notify(state)
	if err != nil {
		log.Error("notify stream data error", err)
		reported = false
	}

	return reported
}

func (s *StreamService) notify(data any) error {
	var err error

	for _, sub := range s.subs {
		err2 := sub.notifier.Notify(sub.subID, data)
		if err == nil {
			err = err2
		}
	}
	return err
}
