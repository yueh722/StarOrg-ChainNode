package nodeproxy

import (
	"chain/rpc/client"
	"errors"
	"fmt"
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

	nodeClient, err := client.New(client.ClientConfig{
		AppName:    cfg.AppName,
		ServerAddr: cfg.ServerAddr,
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
				fmt.Println("Error in ReportNodeStatus:", err)
				needsReconnect = true
			}

			if err := nodeClient.StatusClient.ReportStorageStatus(); isConnectionDead(err) {
				fmt.Println("Error in ReportNodeStorage:", err)
				needsReconnect = true
			}

		case err := <-errCh:
			// subscription error
			if err != nil {
				fmt.Println("Error in SubscribeCommand:", err)
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
