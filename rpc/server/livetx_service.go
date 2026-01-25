package server

import (
	"chain/lib/zlog"
	"chain/rpc/common"
	"chain/rpc/common/notification"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type LiveTxService struct {
	server       *rpc.Server
	cfg          *ServerConfig
	transmitters map[string]bool
	mux          sync.Mutex
}

func NewLiveTxService(server *rpc.Server, cfg *ServerConfig) *LiveTxService {

	c := &LiveTxService{
		server:       server,
		cfg:          cfg,
		transmitters: make(map[string]bool),
	}

	if err := server.RegisterName(common.LiveTxNamespace{}.GetNamesapceString(), c); err != nil {
		zlog.Infof("Failed to register liveTx service: %v", err)
		return nil
	}

	return c
}

func (c *LiveTxService) EnableTransmitter(nodeID string, rpcOnline bool) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	if !rpcOnline {
		return false
	}

	if len(c.transmitters) == 0 {
		zlog.Info("enabled tx transmitter for node: ", nodeID)
		c.transmitters[nodeID] = true
		return true
	}

	if _, ok := c.transmitters[nodeID]; ok {
		//zlog.Info("enabled tx transmitter for node already exist: ", nodeID)
		return true
	}

	return false
}

func (c *LiveTxService) OnNotify(msg *notification.Message) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if msg.Type == notification.MsgTypeOffline {
		nodeID := msg.Payload.(string)
		if _, ok := c.transmitters[nodeID]; ok {
			delete(c.transmitters, nodeID)
			zlog.Info("removed transmitter: ", nodeID)
		}
	}
}
