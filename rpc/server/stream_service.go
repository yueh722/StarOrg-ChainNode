package server

import (
	"chain/lib/queuestore"
	"chain/rpc/commandlog"
	"chain/rpc/common"
	"context"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type customerSub struct {
	notifier *rpc.Notifier
	subID    rpc.ID
}

// StreamService is used to provide live stream data about chain network for webserver.
type StreamService struct {
	subs map[rpc.ID]customerSub
	mut  sync.Mutex
}

func NewStreamService() *StreamService {
	return &StreamService{}
}

func (s *StreamService) GetCustomerCount() int {

	s.mut.Lock()
	defer s.mut.Unlock()

	return len(s.subs)
}

//args
//app
//

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

	switch state := state.(type) {
	case queuestore.PersistedEvent:
	case common.StatusNode:
	case common.StatusStorage:
	case commandlog.CommandResult:
	default:
		log.Println("Unknown state", state)
	}

	err = s.notify(state)
	if err != nil {
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
