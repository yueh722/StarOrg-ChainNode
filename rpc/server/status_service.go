package server

import (
	"chain/rpc/common"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

// SetupService is designed for nodeproxy reportting status about node to NodeMaster.
// Request from: NodeProxy
type StatusService struct {
	mu     sync.Mutex
	stream *StreamService
}

func NewStatusService(stream *StreamService) *StatusService {
	return &StatusService{
		mu:     sync.Mutex{},
		stream: stream,
	}
}

func (s *StatusService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(common.StatusNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register status service: %v", err)
	}

	return nil
}

// Method
func (s *StatusService) ReportNodeStatus(ctx context.Context, status *common.StatusNode) error {
	if s.stream != nil {
		s.stream.ReportState(ctx, status)
	}

	//log.Printf("[NodeStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportStorageStatus(ctx context.Context, status *common.StatusStorage) error {

	if s.stream != nil {
		s.stream.ReportState(ctx, status)
	}

	//zlog.Printf("[StorageStatus] %v\n", status)
	return nil
}
