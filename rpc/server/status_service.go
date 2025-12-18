package server

import (
	"Chain/rpc/common"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type StatusService struct {
	mu sync.Mutex
}

func NewStatusService() *StatusService {
	return &StatusService{
		mu: sync.Mutex{},
	}
}

func (s *StatusService) RegisterService(server *rpc.Server) error {
	if err := server.RegisterName("status", s); err != nil {
		return fmt.Errorf("Failed to register status service:", err)
	}

	return nil
}

func (s *StatusService) ReportNodeStatus(status common.StatusNode) error {

	log.Printf("[NodeStatus] %+v\n", status)
	return nil
}

func (s *StatusService) ReportStorageStatus(status common.StatusStorage) error {

	log.Printf("[StorageStatus] %+v\n", status)
	return nil
}
