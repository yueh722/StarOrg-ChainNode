package nodemaster

import (
	"chain/internal/common"
	"chain/lib/cmdlogstore"
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/rpc/security"
	"chain/rpc/server"
	"chain/utils"
	"errors"
	"path/filepath"
	"sync"
	"time"
)

var (
	wsServer *server.Server

	//
	nodelogStore   queuestore.QueueStore
	setuplogStore  cmdlogstore.CommandLogStore
	logPath        = common.NodeMasterLogPath
	dataPath       = common.NodeMasterAppDataPath
	firewallPolicy security.FirewallPolicy
	muxPolicy      sync.Mutex
)

func init() {
	var err error

	nodelogStore = queuestore.NewInMemoryStore()

	utils.TryCreateDir(dataPath)

	setuplogStore, err = cmdlogstore.NewSqliteCommandLogStore(cmdlogstore.Config{
		Path:        filepath.Join(dataPath, "setuplog.db"),
		Loc:         nil,
		InitTimeout: 3 * time.Second,
		OpTimeout:   5 * time.Second,
		BusyTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("init setuplog.db err: %v", err)
	}
}

// StartServer function won't return until server down or failure.
func StartServer(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	UpdateFirewallPolicy(security.FirewallPolicy{
		FWMode:         cfg.FWMode,
		AllowOnlyHosts: cfg.AllowOnlyHosts,
	})

	wsServer, err := server.NewServer(server.ServerConfig{
		AppName:    cfg.AppName,
		Addr:       cfg.Addr,
		AppDataDir: common.NodeMasterAppDataPath,
		LogDir:     common.NodeMasterLogPath,
		Option: security.SecurityOption{
			//FWMode:         cfg.FWMode,
			//AllowOnlyHosts: cfg.AllowOnlyHosts,
			FetchPolicy: func() security.FirewallPolicy {
				muxPolicy.Lock()
				defer muxPolicy.Unlock()
				return firewallPolicy
			},
			TokenSecret: security.SharedSecret,
		},
		Tls: cfg.Tls,
	})
	if err != nil {
		return err
	}

	return wsServer.Serve()
}

func UpdateFirewallPolicy(policy security.FirewallPolicy) {
	muxPolicy.Lock()
	defer muxPolicy.Unlock()
	firewallPolicy = policy
}
