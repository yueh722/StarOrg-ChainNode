package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func newCommandID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func NewServer(addr string) (*Server, error) {
	//check addr
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid server address: %s", addr)
	}

	server := rpc.NewServer()

	setup := NewSetupService()
	if err := setup.RegisterService(server); err != nil {
		return nil, err
	}

	status := NewStatusService()
	if err := status.RegisterService(server); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle("/ws", server.WebsocketHandler([]string{"*"}))
	mux.Handle("/rpc", server)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "RPC Server is running\n")
	})

	fmt.Printf("WebSocket endpoint: ws://%s:%s/ws", parts[0], parts[1])
	fmt.Printf("HTTP RPC endpoint:  http://%s:%s/rpc", parts[0], parts[1])

	return &Server{
		SetupService:  setup,
		StatusService: status,
		mux:           mux,
		addr:          addr,
	}, nil
}

func (s *Server) Serve() error {
	fmt.Println("Server starting on :", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}
