package nodemaster

import (
	"Chain/lib/queuestore"
	"Chain/rpc/server"
)

var (
	wsServer *server.Server
	logStore queuestore.QueueStore
)

func init() {
	logStore = queuestore.NewInMemoryStore()
}

// block if server running normal.
func StartServer() error {
	var err error

	wsServer, err = server.NewServer(server.ServerConfig{
		Addr:     ":2222",
		LogStore: logStore,
	})
	if err != nil {
		return err
	}

	return wsServer.Serve()
}
