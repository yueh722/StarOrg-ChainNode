package server

import (
	"chain/rpc/common/namespace"
	"chain/rpc/common/types"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

// SetupService is designed for nodeagent reportting status about node to NodeMaster.
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
	if err := server.RegisterName(namespace.StatusNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register status service: %v", err)
	}

	return nil
}

// Method
func (s *StatusService) ReportNodeStatus(ctx context.Context, status *types.StatusNode) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//log.Printf("[NodeStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportStorageStatus(ctx context.Context, status *types.StatusStorage) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//zlog.Printf("[StorageStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportCometbftProcessStatus(ctx context.Context, status *types.StatusCometbftProcess) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//zlog.Printf("[ReportCometbftProcessStatus] %v\n", status)
	return nil
}
