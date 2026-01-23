package server

import (
	"chain/lib"
	log "chain/lib/zlog"
	"chain/rpc/common"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, delegator common.DelegatorInfo) (*rpc.Subscription, error) {

	if !strings.HasPrefix(delegator.App, lib.DelegatorApp) {
		return nil, fmt.Errorf("unknown delegator %v", delegator.App)
	}

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	// use initialization status that stored on master side.
	delegator.StatusNode.Initialized = s.chainState.IsNodeInitialized(delegator.StatusNode.NodeID)

	nodeHostInfoBytes, _ := json.MarshalIndent(delegator.StatusNode.HostInfo, "", "  ")

	log.Printf("Received SubscribeCommand that requests from node, \n app: %s, name: %s, id: %s, initialized: %v, host: %v",
		delegator.App,
		delegator.StatusNode.NodeName,
		delegator.StatusNode.NodeID,
		delegator.StatusNode.Initialized,
		string(nodeHostInfoBytes),
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
	s.NodeNameCache[clientSub.NodeID] = clientSub.StatusNode.NodeName

}

// Method
func (s *SetupService) InitializationComplete(nodeID string, status *common.StatusNode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clientStub, ok := s.subs[nodeID]; ok {
		if status.NodeID == nodeID {
			clientStub.StatusNode = *status
			log.Infof("node initialization complete, nodeID: %v, nodeName: %v", status.NodeID, status.NodeName)
		}
	}
}
