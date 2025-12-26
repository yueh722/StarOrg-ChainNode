package server

import (
	"chain/rpc/common"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type NodeStatusCallback func(status *common.StatusNode)
type NodeStorageCallback func(status *common.StatusStorage)

// SetupService is designed for nodeproxy reportting status about node to NodeMaster.
// Request from: NodeProxy
type StatusService struct {
	mu        sync.Mutex
	fnStatus  NodeStatusCallback
	fnStorage NodeStorageCallback
}

func NewStatusService() *StatusService {
	return &StatusService{
		mu:        sync.Mutex{},
		fnStatus:  nil,
		fnStorage: nil,
	}
}

func (s *StatusService) SetCallback(fnStatus NodeStatusCallback, fnStorage NodeStorageCallback) {
	s.fnStatus = fnStatus
	s.fnStorage = fnStorage
}

func (s *StatusService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("status", s); err != nil {
		return fmt.Errorf("Failed to register status service:", err)
	}

	return nil
}

func (s *StatusService) GetServiceName() string {
	return "status"
}

// Method
func (s *StatusService) ReportNodeStatus(status *common.StatusNode) error {

	if s.fnStatus != nil {
		s.fnStatus(status)
	}

	log.Printf("[NodeStatus] %v\n", status)
	return nil
}

// Method
func (s *StatusService) ReportStorageStatus(status *common.StatusStorage) error {

	if s.fnStorage != nil {
		s.fnStorage(status)
	}

	log.Printf("[StorageStatus] %v\n", status)
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
