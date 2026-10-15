package server

import (
	"chain/rpc/common/namespace"
	"chain/rpc/common/setupTypes"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

// SetupService is designed for nodeagent reportting status about node to NodeMaster.
// Request from: NodeProxy
type StatusService struct {
	mu           sync.Mutex
	stream       *StreamService
	storageStore map[string]*setupTypes.StatusStorage
}

func NewStatusService(stream *StreamService) *StatusService {
	return &StatusService{
		mu:           sync.Mutex{},
		stream:       stream,
		storageStore: make(map[string]*setupTypes.StatusStorage),
	}
}

func (s *StatusService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName(namespace.StatusNamespace{}.GetNamesapceString(), s); err != nil {
		return fmt.Errorf("Failed to register status service: %v", err)
	}

	return nil
}

// Method
func (s *StatusService) ReportNodeStatus(ctx context.Context, status *setupTypes.StatusNode) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//log.Printf("[NodeStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportStorageStatus(ctx context.Context, status *setupTypes.StatusStorage) error {

	s.mu.Lock()
	if s.storageStore == nil {
		s.storageStore = make(map[string]*setupTypes.StatusStorage)
	}
	if status.NodeID != "" {
		s.storageStore[status.NodeID] = status
	}
	if status.NodeName != "" {
		s.storageStore[status.NodeName] = status
	}
	s.mu.Unlock()

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	return nil
}

// Method: query all nodes' latest reported storage status
func (s *StatusService) GetStorageStatus(ctx context.Context) (map[string]*setupTypes.StatusStorage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make(map[string]*setupTypes.StatusStorage)
	for k, v := range s.storageStore {
		res[k] = v
	}
	return res, nil
}

// Method
func (s *StatusService) ReportCometbftProcessStatus(ctx context.Context, status *setupTypes.StatusCometbftProcess) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//zlog.Printf("[ReportCometbftProcessStatus] %v\n", status)
	return nil
}
