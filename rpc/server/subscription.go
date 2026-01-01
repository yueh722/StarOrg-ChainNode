package server

import (
	"chain/lib"
	"chain/rpc/common"
	"context"
	"log"

	"github.com/ethereum/go-ethereum/rpc"
)

type DelegatorInfo struct {
	App       string
	NodeState int
}

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, statusNode common.StatusNode, delegator DelegatorInfo) (*rpc.Subscription, error) {

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	log.Println("Received SubscribeCommand request, node:", statusNode.NodeName)

	nodeInfo := statusNode.NodeInfo
	if nodeInfo.NodeID == "" {
		//this is an uninitialized node.
		//assign a node ID to it
		nodeInfo.NodeID = lib.NewNodeId()
	}

	sub := notifier.CreateSubscription()

	s.AddOrUpdateSubscription(&ClientSub{
		Notifier:   notifier,
		NodeID:     nodeInfo.NodeID,
		StatusNode: statusNode,
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
