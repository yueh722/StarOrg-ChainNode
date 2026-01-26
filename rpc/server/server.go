package server

import (
	"chain/lib/cmdlogstore"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/common/notification"
	"chain/rpc/security"
	"chain/rpc/server/middleware"
	"chain/utils"
	"crypto/rand"
	"encoding/hex"
	"errors"
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

/*
type Ticker struct {
	NotifierMgr *notification.Manager
}

func NewTicker(rpcServer *rpc.Server, listeners ...notification.Listener) *Ticker {

	t := &Ticker{
		NotifierMgr: notification.NewManager(listeners...),
	}
	if err := rpcServer.RegisterName(common.TickerNamespace{}.GetNamesapceString(), t); err != nil {
		log.Infof("Failed to register ticker service: %v", err)
		return nil
	}

	return t
}

func (t *Ticker) Start(ctx context.Context) {

	cc, ok := ctx.Value(common.KeyClientID).(*common.ClientContext)
	if !ok {
		return
	}

	log.Info("Ticker incoming: clientID: ", cc.ClientID, " Token: ", cc.Token)
	client, ok := rpc.ClientFromContext(ctx)
	if !ok {
		return
	}

	err := client.Call(nil, common.GetMethodName(""), &common.KeepUntilDead{
		ServerID: "master",
		ClientID: cc.ClientID,
		SeqNum:   cc.SeqNum + 1,
	})
	if err != nil {
		log.Infof("KeepUntilDead returned from client, connection might closed. err: %v", err)
	}
}

*/

type Server struct {
	SetupService  *SetupService
	StatusService *StatusService
	mux           *http.ServeMux
	addr          string
	srv           *http.Server
	tls           TlsOption
	RpcServer     *rpc.Server
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
		return nil, fmt.Errorf("invalid Server address: %s", cfg.Addr)
	}

	rpcServer := rpc.NewServer()

	stream := NewStreamService()
	if err := stream.RegisterService(rpcServer); err != nil {
		return nil, err
	}

	liveTx := NewLiveTxService(rpcServer, &cfg, stream)
	if liveTx == nil {
		return nil, errors.New("failed to create livetx service")
	}

	status := NewStatusService(stream)
	if err := status.RegisterService(rpcServer); err != nil {
		return nil, err
	}

	logSrv := NewNodeLogService(queuestore.NewInMemoryStore(), stream)
	if err := logSrv.RegisterService(rpcServer); err != nil {
		return nil, err
	}

	setuplogStore, _ := NewSetuplogStore(filepath.Join(cfg.AppDataDir, "setuplog.db"))
	setup := NewSetupService(setuplogStore, stream, notification.NewManager(liveTx))
	if err := setup.RegisterService(rpcServer); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.Handle("/websocket", rpcServer.WebsocketHandler([]string{"*"}))
	mux.Handle("/rpc", rpcServer)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "RPC Server is running")
	})

	middlewares := []middleware.Middleware{
		security.NewFirewall(cfg.Option).Middleware,
		security.NewAuth(cfg.Option.TokenSecret).Middleware,
	}

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: middleware.Chain(mux, middlewares...),
	}

	return &Server{
		SetupService:  setup,
		StatusService: status,
		mux:           mux,
		addr:          cfg.Addr,
		srv:           srv,
		tls:           cfg.Tls,
		RpcServer:     rpcServer,
	}, nil
}

func (s *Server) CheckTlsOption() bool {

	if s.tls.TlsCertFile != "" && s.tls.TlsKeyFile != "" {
		if fileutil.IsExist(s.tls.TlsCertFile) && fileutil.IsExist(s.tls.TlsKeyFile) {
			return true
		}

		log.Println("cert and key files are not exist. ignore tls option")
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
		log.Println("Server starting on ", s.addr, " with TLS")
		return s.srv.ListenAndServeTLS(s.tls.TlsCertFile, s.tls.TlsKeyFile)
	} else {
		protocol = "http"
		log.Println("Server starting on ", s.addr)
		return s.srv.ListenAndServe()
	}
}
