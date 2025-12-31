package server

import (
	"chain/lib/logdb"
	"chain/lib/queuestore"
	"chain/rpc/security"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type TlsOption struct {
	TlsCertFile string
	TlsKeyFile  string
}

type Server struct {
	SetupService  *SetupService
	StatusService *StatusService
	mux           *http.ServeMux
	addr          string
	srv           *http.Server
	tls           TlsOption
}

type ServerConfig struct {
	AppName       string
	Addr          string
	NodelogStore  queuestore.QueueStore
	SetuplogStore logdb.CommandLogStore
	Option        security.SecurityOption
	Tls           TlsOption
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

	stream := NewStreamService()

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

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: security.NewFirewall(cfg.Option).Wrap(mux),
	}

	return &Server{
		SetupService:  setup,
		StatusService: status,
		mux:           mux,
		addr:          cfg.Addr,
		srv:           srv,
		tls:           cfg.Tls,
	}, nil
}

func (s *Server) CheckTlsOption() bool {

	if s.tls.TlsCertFile != "" && s.tls.TlsKeyFile != "" {
		if fileutil.IsExist(s.tls.TlsCertFile) && fileutil.IsExist(s.tls.TlsKeyFile) {
			return true
		}

		log.Println("crt and key files are not exist. ignore tls option")
	}

	/*
		_, err := tls.LoadX509KeyPair(s.tls.TlsCertFile, s.tls.TlsKeyFile)
		if err != nil {
			log.Println("failed to load certFile and keyFile, err:", err)
			return false
		}
	*/

	return false
}

func (s *Server) Serve() error {
	var protocol string
	var wsprotocol string

	tlsEnabled := s.CheckTlsOption()
	if tlsEnabled {
		wsprotocol = "wss"
		protocol = "https"
	} else {
		wsprotocol = "ws"
		protocol = "http"
	}

	parts := strings.SplitN(s.addr, ":", 2)
	localAddr := net.JoinHostPort("localhost", parts[1])
	log.Printf("WebSocket endpoint: %s://%s/ws\n", wsprotocol, localAddr)
	log.Printf("RPC endpoint:  %s://%s/rpc\n", protocol, localAddr)
	log.Printf("HTTP endpoint:  %s://%s\n", protocol, localAddr)

	if tlsEnabled {
		protocol = "https"
		log.Println("Server starting on", s.addr, "with TLS")
		return s.srv.ListenAndServeTLS(s.tls.TlsCertFile, s.tls.TlsKeyFile)
	} else {
		protocol = "http"
		log.Println("Server starting on", s.addr)
		return s.srv.ListenAndServe()
	}
}
