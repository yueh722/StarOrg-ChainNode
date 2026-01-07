package nodeproxy

import (
	"chain/internal/common"
	"chain/lib"
	"chain/rpc/client"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

func init() {

}

func getNodeBaseDir(nodeDir, nodeID string) string {
	return filepath.Join(nodeDir, nodeID)
}

func StartClient() error {

	cfg, err := GetConfig()
	if err != nil {
		return err
	}

	nodeInfo, err := lib.GetNodeInfo(cfg.HomeDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		//configuration data error.
		return fmt.Errorf("GetNodeInfo error: %v", err)
	}

	if nodeInfo == nil {
		// init new node id if current node is uninitialized
		if nodeInfo, err = lib.InitializeNewNodeID(cfg.HomeDir); err != nil {
			return fmt.Errorf("failed to initialize new nodeID: %v", err)
		}
	}

	nodeClient, err := client.New(client.ClientConfig{
		AppName:     cfg.AppName,
		ServerAddr:  cfg.ServerAddr,
		HomeDir:     cfg.HomeDir,
		NodeBaseDir: getNodeBaseDir(cfg.HomeDir, nodeInfo.NodeID),
		LogDir:      common.NodeProxyLogPath,
		AppDataDir:  common.NodeProxyAppDataPath,
		NodeInfo:    nodeInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
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
	if err == nil {
		return false
	}

	errText := err.Error()
	if strings.Contains(errText, "Bad Request") ||
		strings.Contains(errText, "tls: first record does not look like a TLS handshake") {
		return true
	}

	if errors.Is(err, rpc.ErrClientQuit) {
		return true
	}

	return false
}
