package server

import (
	log "chain/lib/zlog"
	"chain/rpc/common"
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, delegator common.DelegatorInfo) (*rpc.Subscription, error) {

	if !strings.HasPrefix(delegator.App, common.DelegatorApp) {
		return nil, fmt.Errorf("unknown delegator %v", delegator.App)
	}

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	// use initialization status that stored on master side.
	delegator.StatusNode.Initialized = s.chainState.IsNodeInitialized(delegator.StatusNode.NodeID)

	log.Printf("Received SubscribeCommand request from node, app %s, id: %s, name: %s, initialized: %v",
		delegator.App,
		delegator.StatusNode.NodeID,
		delegator.StatusNode.NodeName,
		delegator.StatusNode.Initialized,
	)

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

	log.Info("add subscription:", clientSub.NodeID, clientSub.Notifier, clientSub.SubID)
}
