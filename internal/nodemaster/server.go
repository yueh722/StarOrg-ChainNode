package nodemaster

import (
	"chain/internal/common"
	"chain/lib/logdb"
	"chain/lib/queuestore"
	"chain/rpc/server"
	"log"
	"path"
	"time"
)

var (
	wsServer *server.Server

	//
	nodelogStore  queuestore.QueueStore
	setuplogStore logdb.CommandLogStore
	logPath       = common.NodeMasterLogPath
	dataPath      = common.NodeMasterAppDataPath
)

func init() {
	nodelogStore = queuestore.NewInMemoryStore()

	var err error
	setuplogStore, err = logdb.NewSqliteCommandLogStore(logdb.Config{
		Path:        path.Join(dataPath, "setuplog.db"),
		Loc:         nil,
		InitTimeout: 3 * time.Second,
		OpTimeout:   5 * time.Second,
		BusyTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// StartServer function won't return until server down or failure.
func StartServer() error {

	cfg, err := GetConfig()
	if err != nil {
		return err
	}

	wsServer, err = server.NewServer(server.ServerConfig{
		AppName:       cfg.AppName,
		Addr:          cfg.Addr,
		NodelogStore:  nodelogStore,
		SetuplogStore: setuplogStore,
	})
	if err != nil {
		return err
	}

	return wsServer.Serve()
}
