package server

import (
	"chain/lib/cmdlogstore"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/security"
	"chain/utils"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
	AppName    string
	Addr       string
	AppDataDir string
	LogDir     string
	Option     security.SecurityOption
	Tls        TlsOption
}

func newCommandID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func NewSetuplogStore(dataFile string) (cmdlogstore.CommandLogStore, error) {
	utils.TryCreateDir(filepath.Dir(dataFile))

	setuplogStore, err := cmdlogstore.NewSqliteCommandLogStore(cmdlogstore.Config{
		Path:        dataFile,
		Loc:         nil,
		InitTimeout: 3 * time.Second,
		OpTimeout:   5 * time.Second,
		BusyTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Panicf("init setuplogstore failure, err: %v", err)
	}

	return setuplogStore, nil
}

func NewServer(cfg ServerConfig) (*Server, error) {
	//check addr
	parts := strings.SplitN(cfg.Addr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid server address: %s", cfg.Addr)
	}

	server := rpc.NewServer()

	stream := NewStreamService()
	if err := stream.RegisterService(server); err != nil {
		return nil, err
	}

	//transmitLiveTransaction(stream)

	setuplogStore, _ := NewSetuplogStore(filepath.Join(cfg.AppDataDir, "setuplog.db"))
	setup := NewSetupService(setuplogStore, stream)
	if err := setup.RegisterService(server); err != nil {
		return nil, err
	}

	status := NewStatusService(stream)
	if err := status.RegisterService(server); err != nil {
		return nil, err
	}

	logSrv := NewNodeLogService(queuestore.NewInMemoryStore(), stream)
	if err := logSrv.RegisterService(server); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle("/ws", server.WebsocketHandler([]string{"*"}))
	mux.Handle("/rpc", server)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "RPC Server is running")
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
		// valid tsl files
			_, err := tls.LoadX509KeyPair(s.tls.TlsCertFile, s.tls.TlsKeyFile)
			if err != nil {
				zlog.Println("failed to load certFile and keyFile, err:", err)
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
	log.Printf("WebSocket endpoint: %s://%s/ws", wsprotocol, localAddr)
	log.Printf("RPC endpoint:  %s://%s/rpc", protocol, localAddr)
	log.Printf("HTTP endpoint:  %s://%s", protocol, localAddr)

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
