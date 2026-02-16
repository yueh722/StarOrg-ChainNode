package server

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/common/namespace"
	"chain/rpc/common/notification"
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type LiveTxService struct {
	server       *rpc.Server
	cfg          *ServerConfig
	transmitters map[string]bool
	mux          sync.Mutex
	stream       *StreamService
}

func NewLiveTxService(server *rpc.Server, cfg *ServerConfig, stream *StreamService) *LiveTxService {

	s := &LiveTxService{
		server:       server,
		cfg:          cfg,
		transmitters: make(map[string]bool),
		stream:       stream,
	}

	if err := server.RegisterName(namespace.LiveTxNamespace{}.GetNamesapceString(), s); err != nil {
		zlog.Infof("Failed to register liveTx service: %v", err)
		return nil
	}

	return s
}

func (s *LiveTxService) EnableTransmitter(nodeID string, rpcOnline bool) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	if !rpcOnline {
		return false
	}

	if len(s.transmitters) == 0 {
		zlog.Info("enabled tx transmitter for node: ", nodeID)
		s.transmitters[nodeID] = true
		return true
	}

	if _, ok := s.transmitters[nodeID]; ok {
		//zlog.Info("enabled tx transmitter for node already exist: ", nodeID)
		return true
	}

	return false
}

func (s *LiveTxService) OnNotify(msg *notification.Message) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if msg.Type == notification.MsgTypeOffline {
		nodeID := msg.Payload.(string)
		if _, ok := s.transmitters[nodeID]; ok {
			delete(s.transmitters, nodeID)
			zlog.Info("removed transmitter: ", nodeID)
		}
	}
}

// Method
func (s *LiveTxService) ReportLiveTx(ctx context.Context, nodeID string, status *lib.StatusTransaction) error {
	s.mux.Lock()
	if _, ok := s.transmitters[nodeID]; !ok {
		s.mux.Unlock()
		return errors.New("received live transaction from illegal transmitter")
	}
	s.mux.Unlock()

	if s.cfg.OutputLiveTx {
		zlog.Infof("received live transaction:\n nodeID: %v, tx: %v", nodeID, status)
	}

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	return nil
}
