package server

import (
	"chain/rpc/common"
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
	if err := server.RegisterName("status", s); err != nil {
		return fmt.Errorf("Failed to register status service: %v", err)
	}

	return nil
}

func (s *StatusService) GetServiceName() string {
	return "status"
}

// Method
func (s *StatusService) ReportNodeStatus(status *common.StatusNode) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//log.Printf("[NodeStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportStorageStatus(status *common.StatusStorage) error {

	if s.stream != nil {
		s.stream.ReportState(status)
	}

	//log.Printf("[StorageStatus] %v\n", status)
	return nil
}

func convertBytesTraditional(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TB", float64(size)/(1024*1024*1024*1024))
	}
}
