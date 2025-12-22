package server

import (
	"Chain/lib/queuestore"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
)

type LogService struct {
	store  queuestore.QueueStore
	broker *queuestore.Broker
}

func NewLogService(store queuestore.QueueStore, broker *queuestore.Broker) *LogService {
	return &LogService{store: store, broker: broker}
}

func (s *LogService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("log", s); err != nil {
		return fmt.Errorf("Failed to register log service:", err)
	}

	return nil
}

type SubscribeArgs struct {
	Since string `json:"since,omitempty"`
}

// SubscribeErrors：JSON-RPC subscription
// client call method: "log_subscribeErrors"
func (s *LogService) SubscribeErrors(ctx context.Context, args SubscribeArgs) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()

	ch := s.broker.Subscribe(256)

	// notify all events that had not sent.
	if args.Since == "" || args.Since == "unsent" {
		_ = s.store.ScanUnsent(0, func(e queuestore.PersistedEvent) error {
			notifier.Notify(sub.ID, e)
			return nil
		})
	}

	//create go routine for notifying new event.
	go func() {
		defer s.broker.Unsubscribe(ch)
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

// Ack：client call Ack to confirm that event has received. method: "log_ack", params: [event_id]
func (s *LogService) Ack(eventID string) (bool, error) {
	if err := s.store.MarkSent(eventID); err != nil {
		return false, err
	}
	return true, nil
}
