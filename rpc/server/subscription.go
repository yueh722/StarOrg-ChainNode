package server

import (
	"chain/lib"
	"context"
	"log"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/ethereum/go-ethereum/rpc"
)

type DelegatorInfo struct {
	App       string
	NodeState int
}

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, nodeInfo lib.NodeInfo, delegator DelegatorInfo) (*rpc.Subscription, error) {

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	log.Println("Received SubscribeCommand request, node:", nodeInfo.NodeName)

	if nodeInfo.NodeID == "" {
		//this is an uninitialized node.
		//assign a node ID to it
		nodeInfo.NodeID = lib.NewNodeId()
	}

	sub := notifier.CreateSubscription()

	s.AddOrUpdateSubscription(sub.ID, &ClientSub{
		Notifier: notifier,
		NodeID:   nodeInfo.NodeID,
		NodeInfo: nodeInfo,
	})

	go func() {
		<-sub.Err()
		s.mu.Lock()
		delete(s.subs, sub.ID)
		s.mu.Unlock()
	}()

	return sub, nil
}

func (s *SetupService) AddOrUpdateSubscription(subID rpc.ID, clientSub *ClientSub) {

	s.mu.Lock()
	defer s.mu.Unlock()

	// find subscription of current node that already exist.
	notifiers := maputil.Filter(s.subs, func(_ rpc.ID, sub *ClientSub) bool {
		return sub.NodeID == clientSub.NodeID
	})

	for removeID, _ := range notifiers {
		delete(s.subs, removeID)
	}

	s.subs[subID] = clientSub
}
