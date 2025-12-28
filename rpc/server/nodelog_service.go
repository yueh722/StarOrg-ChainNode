package server

import (
	"chain/lib/queuestore"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
)

//
// NodeLogService is designed for webserver getting notification from cometbft-node and cometbft-app.
// Request from: WebServer

type NodeLogService struct {
	nodeLogStore queuestore.QueueStore
	stream       *StreamService
	unsendLogs   map[string]queuestore.PersistedEvent
	mut          sync.Mutex
}

func NewNodeLogService(store queuestore.QueueStore, stream *StreamService) *NodeLogService {

	service := &NodeLogService{
		nodeLogStore: store,
		stream:       stream,
	}

	_ = service.nodeLogStore.ScanUnsent(0, func(e queuestore.PersistedEvent) error {
		service.unsendLogs[e.EventID] = e
		return nil
	})

	go service.PublishLog()

	return service
}

func (s *NodeLogService) PublishLog() {

	for {
		if s.stream.GetCustomerCount() == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		var sendingLogs = make(map[string]queuestore.PersistedEvent)
		s.mut.Lock()
		sendingLogs = maputil.Merge(s.unsendLogs, sendingLogs)
		s.mut.Unlock()

		for eventID, unsendLog := range sendingLogs {
			ok := s.stream.ReportState(unsendLog)
			if ok {
				s.mut.Lock()
				delete(s.unsendLogs, eventID)
				s.mut.Unlock()
			}
		}
	}
}

func (s *NodeLogService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("nodelog", s); err != nil {
		return fmt.Errorf("Failed to register nodelog service:", err)
	}

	return nil
}

type SubscribeArgs struct {
	Since string `json:"since,omitempty"`
}

// SubscribeErrors：JSON-RPC subscription
// Subscriber call method: "nodelog_subscribeErrors"
func (s *NodeLogService) _SubscribeErrors(ctx context.Context, args SubscribeArgs) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()

	ch := s.nodeLogStore.GetBroker().Subscribe(256)

	// notify all events that had not sent.
	if args.Since == "" || args.Since == "unsent" {
		_ = s.nodeLogStore.ScanUnsent(0, func(e queuestore.PersistedEvent) error {
			notifier.Notify(sub.ID, e)
			return nil
		})
	}

	//create go routine for notifying new event.
	go func() {
		defer s.nodeLogStore.GetBroker().Unsubscribe(ch)
		for {
			select {
			case <-sub.Err(): // client disconnected
				return
			case e, ok := <-ch:
				if !ok {
					return
				}
				notifier.Notify(sub.ID, e)
			}
		}
	}()

	return sub, nil
}

// Ack：Subscriber call Ack to confirm that event has received.
// method: "nodelog_ack", params: [event_id]
func (s *NodeLogService) Ack(eventID string) (bool, error) {
	if err := s.nodeLogStore.MarkSent(eventID); err != nil {
		return false, err
	}
	return true, nil
}

// Push new event
// Nodeproxy call method: "nodelog_reportLog"
func (s *NodeLogService) ReportLog(e queuestore.PersistedEvent) error {

	s.mut.Lock()
	s.unsendLogs[e.EventID] = e
	s.mut.Unlock()

	//notify receiver to receive error log from node
	return s.nodeLogStore.Put(e)
}
