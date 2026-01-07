package server

import (
	"chain/rpc/common"
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/rpc"
)

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, delegator common.DelegatorInfo) (*rpc.Subscription, error) {

	if delegator.App != common.DelegatorApp {
		return nil, fmt.Errorf("unknown delegator %v", delegator.App)
	}

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	log.Printf("Received SubscribeCommand request from node, id: %s, name: %s\n",
		delegator.StatusNode.NodeID,
		delegator.StatusNode.NodeName)

	nodeInfo := delegator.StatusNode.NodeInfo
	if nodeInfo.NodeID == "" {
		return nil, rpc.ErrNotificationsUnsupported
	}

	sub := notifier.CreateSubscription()

	s.AddOrUpdateSubscription(&ClientSub{
		Notifier:   notifier,
		NodeID:     nodeInfo.NodeID,
		StatusNode: delegator.StatusNode,
		SubID:      sub.ID,
	})

	go func() {
		<-sub.Err()
		s.mu.Lock()
		delete(s.subs, nodeInfo.NodeID)
		s.mu.Unlock()
	}()

	return sub, nil
}

func (s *SetupService) AddOrUpdateSubscription(clientSub *ClientSub) {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.subs[clientSub.NodeID] = clientSub
}
