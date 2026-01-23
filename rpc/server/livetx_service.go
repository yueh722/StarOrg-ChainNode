package server

import (
	"chain/lib/zlog"
	"chain/rpc/common"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type LiveTxService struct {
	server       *rpc.Server
	cfg          *ServerConfig
	transmitters map[string]bool
	mux          sync.Mutex
}

func newLiveTxService(server *rpc.Server, cfg *ServerConfig) *LiveTxService {

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

func (c *LiveTxService) Enabled(nodeID string, rpcOnline bool) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	if len(c.transmitters) == 0 {

		return true
	}

	return true
}
