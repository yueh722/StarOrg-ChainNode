package server

import (
	"Chain/rpc/common"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type NodeStatusCallback func(status *common.StatusNode)
type NodeStorageCallback func(status *common.StatusStorage)

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

func (s *StatusService) ReportNodeStatus(status *common.StatusNode) error {

	if s.fnStatus != nil {
		s.fnStatus(status)
	}

	log.Printf("[NodeStatus] %v\n", status)
	return nil
}

func (s *StatusService) ReportStorageStatus(status *common.StatusStorage) error {

	if s.fnStorage != nil {
		s.fnStorage(status)
	}

	log.Printf("[StorageStatus] %v\n", status)
	return nil
}
