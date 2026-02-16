package server

import (
	"chain/lib"
	log "chain/lib/zlog"
	"chain/rpc/common/notification"
	"chain/rpc/common/setupTypes"
	"chain/utils"
	"context"
	"fmt"
	"strings"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/ethereum/go-ethereum/rpc"
)

// Subscription
func (s *SetupService) SubscribeCommand(ctx context.Context, delegator setupTypes.DelegatorInfo) (*rpc.Subscription, error) {

	if !strings.HasPrefix(delegator.App, lib.DelegatorApp) {
		return nil, fmt.Errorf("unknown delegator %v", delegator.App)
	}

	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}

	client, ok := rpc.ClientFromContext(ctx)
	if !ok {
		log.Info("rpc client from context is not available: ", delegator.StatusNode.NodeName)
	}

	// use initialization status that stored on master side.
	delegator.StatusNode.Initialized = s.chainState.IsNodeInitialized(delegator.StatusNode.NodeID)
	delegator.StatusNode.Online = true

	// filter private ip address for debugging purpose
	HostIP, err := utils.FindIPByPrefix(delegator.StatusNode.HostInfo.IpAddrs, "192.168.1")
	if err != nil {
		HostIP = strings.Join(delegator.StatusNode.HostInfo.IpAddrs, ",")
	}
	//nodeHostInfoBytes, _ := json.MarshalIndent(delegator.StatusNode.HostInfo, "", "  ")

	log.Printf("Received SubscribeCommand requested from node, app: %s, name: %s, id: %s, initialized: %v, host: %v",
		delegator.App,
		delegator.StatusNode.NodeName,
		delegator.StatusNode.NodeID,
		delegator.StatusNode.Initialized,
		string(HostIP),
	)

	nodeInfo := delegator.StatusNode.NodeInfo
	if nodeInfo.NodeID == "" {
		return nil, rpc.ErrNotificationsUnsupported
	}

	sub := notifier.CreateSubscription()

	if err := s.AddOrUpdateSubscription(&ClientSub{
		Notifier:   notifier,
		Client:     client,
		NodeID:     nodeInfo.NodeID,
		DeviceID:   delegator.DeviceID,
		StatusNode: delegator.StatusNode,
		SubID:      sub.ID,
	}); err != nil {
		log.Println("add subscription error: ", err)
		return nil, err
	}

	//notify stream's subscribers that node is online
	s.broadcaseClientStatusChanged(nodeInfo.NodeID, true)

	go func() {
		err := <-sub.Err()
		log.Info("rpc client subscription closed,", err)

		//if err := s.PingClient(notifier, sub.ID, nodeInfo.NodeID); err != nil {}
		s.broadcaseClientStatusChanged(nodeInfo.NodeID, false)

		s.mu.Lock()
		delete(s.subs, nodeInfo.NodeID)
		s.mu.Unlock()
	}()

	return sub, nil
}

func (s *SetupService) AddOrUpdateSubscription(clientSub *ClientSub) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existSub, ok := s.subs[clientSub.NodeID]; ok {
		if existSub.DeviceID != clientSub.DeviceID {
			return fmt.Errorf("DeviceID conflict (new deviceID: %s, exist DeviceID: %s) for nodeName %s",
				clientSub.DeviceID, existSub.DeviceID, clientSub.StatusNode.NodeName)
		}
	}

	s.subs[clientSub.NodeID] = clientSub
	s.NodeNameCache[clientSub.NodeID] = clientSub.StatusNode.NodeName
	return nil
}

func (s *SetupService) buildPingCmd(nodeID string) *setupTypes.CommandEnvelope {

	ping := &setupTypes.SetupPing{
		NodeID:    nodeID,
		Timestamp: time.Now().UnixNano(),
	}

	payload, _ := cmtjson.Marshal(ping)

	return &setupTypes.CommandEnvelope{
		CommandID: utils.UUID(),
		Cmd:       setupTypes.CmdPing,
		NodeID:    nodeID,
		Payload:   payload,
		IssuedAt:  utils.NowString(),
	}
}

func (s *SetupService) broadcaseClientStatusChanged(nodeID string, online bool) {

	var msgType string
	switch online {
	case true:
		msgType = notification.MsgTypeOnline
	case false:
		msgType = notification.MsgTypeOffline
	}

	s.notifierMgr.Notify(notification.Message{
		Type:    msgType,
		Payload: nodeID,
		Time:    time.Now().UnixNano(),
	})
}

func (s *SetupService) broadcaseNodeStatusChanged(nodeID string, initialized bool) {

	var msgType string
	switch initialized {
	case true:
		msgType = notification.MsgTypeInitialized
	case false:
		msgType = notification.MsgTypeUnInitialized
	}

	s.notifierMgr.Notify(notification.Message{
		Type:    msgType,
		Payload: nodeID,
		Time:    time.Now().UnixNano(),
	})
}

func (s *SetupService) PingClient(notifier *rpc.Notifier, subID rpc.ID, nodeID string) error {
	err := notifier.Notify(subID, s.buildPingCmd(nodeID))
	if err != nil {
		log.Info("notifier notify err. ", err)
		log.Infof("node %v went offline", s.FormatName(nodeID))
		// client is offline
		return err
	}
	return nil
}

// Method
func (s *SetupService) InitializationComplete(nodeID string, status *setupTypes.StatusNode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clientStub, ok := s.subs[nodeID]; ok {
		if status.NodeID == nodeID {
			clientStub.StatusNode = *status
			clientStub.StatusNode.Online = true
			log.Infof("node initialization complete, nodeID: %v, nodeName: %v", status.NodeID, status.NodeName)
		}
	}
}
