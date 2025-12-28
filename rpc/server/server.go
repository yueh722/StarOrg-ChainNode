package server

import (
	"chain/lib/logdb"
	"chain/lib/queuestore"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"
)

type Server struct {
	SetupService  *SetupService
	StatusService *StatusService
	mux           *http.ServeMux
	addr          string
}

type ServerConfig struct {
	AppName       string
	Addr          string
	NodelogStore  queuestore.QueueStore
	SetuplogStore logdb.CommandLogStore
}

func newCommandID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func NewServer(cfg ServerConfig) (*Server, error) {
	//check addr
	parts := strings.SplitN(cfg.Addr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid server address: %s", cfg.Addr)
	}

	server := rpc.NewServer()

	stream := NewSteamService()

	setup := NewSetupService(cfg.SetuplogStore, stream)
	if err := setup.RegisterService(server); err != nil {
		return nil, err
	}

	status := NewStatusService(stream)
	if err := status.RegisterService(server); err != nil {
		return nil, err
	}

	logSrv := NewNodeLogService(cfg.NodelogStore, stream)
	if err := logSrv.RegisterService(server); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle("/ws", server.WebsocketHandler([]string{"*"}))
	mux.Handle("/rpc", server)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "RPC Server is running\n")
	})

	localAddr := net.JoinHostPort("localhost", parts[1])
	log.Printf("WebSocket endpoint: ws://%s/ws\n", localAddr)
	log.Printf("HTTP RPC endpoint:  http://%s/rpc\n", localAddr)
	log.Printf("HTTP endpoint:  http://%s\n", localAddr)

	return &Server{
		SetupService:  setup,
		StatusService: status,
		mux:           mux,
		addr:          cfg.Addr,
	}, nil
}

func (s *Server) Serve() error {
	log.Println("Server starting on", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}
