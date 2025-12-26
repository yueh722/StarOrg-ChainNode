package nodeproxy

import (
	"chain/lib"
	"chain/rpc/client"
	"errors"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

func init() {

}

func StartClient() error {

	cfg, err := GetConfig()
	if err != nil {
		return err
	}

	nodeInfo, err := lib.GetNodeInfo(cfg.NodeDir)
	if err != nil {
		return err
	}

	nodeClient, err := client.New(client.ClientConfig{
		AppName:     cfg.AppName,
		ServerAddr:  cfg.ServerAddr,
		NodeBaseDir: cfg.NodeDir,
		NodeInfo:    nodeInfo,
	})
	if err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	errCh := nodeClient.SetupClient.SubscribeCommand()

	needsReconnect := false

	for {
		select {
		case <-ticker.C:
			if err := nodeClient.StatusClient.ReportNodeStatus(); isConnectionDead(err) {
				log.Println("Error in ReportNodeStatus:", err)
				needsReconnect = true
			}

			if err := nodeClient.StatusClient.ReportStorageStatus(); isConnectionDead(err) {
				log.Println("Error in ReportNodeStorage:", err)
				needsReconnect = true
			}

		case err := <-errCh:
			if err != nil {
				log.Println("Error in SubscribeCommand:", err)
				needsReconnect = true
			}

		}

		if needsReconnect {
			nodeClient.Reconnect()
			needsReconnect = false
			errCh = nodeClient.SetupClient.SubscribeCommand()
		}
	}
}

func isConnectionDead(err error) bool {
	if errors.Is(err, rpc.ErrClientQuit) {
		return true
	}
	return false
}
