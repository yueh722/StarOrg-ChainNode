package nodeagent

import (
	"chain/internal/common"
	"chain/lib"
	log "chain/lib/zlog"
	"chain/rpc/client"
	"context"
	"errors"
	"fmt"
	"io/fs"
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

func StartClient(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	var initialized bool

	nodeInfo, err := lib.GetNodeInfo(cfg.HomeDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		//configuration data error.
		return fmt.Errorf("GetNodeInfo error: %v", err)
	}

	if nodeInfo == nil {
		initialized = false
		// init new node id if current node is uninitialized
		if nodeInfo, err = lib.InitializeNewNodeID(cfg.HomeDir, cfg.AppName); err != nil {
			return fmt.Errorf("failed to initialize new nodeID: %v", err)
		}
		log.Infof("initialized new node, nodeName: %s, nodeID: %v", nodeInfo.NodeName, nodeInfo.NodeID)
	} else {
		initialized = lib.IsNodeInitialized(cfg.HomeDir, nodeInfo)
		log.Infof("loaded node, nodeName: %s, nodeID: %v", nodeInfo.NodeName, nodeInfo.NodeID)
	}

	nodeClient, err := client.New(&client.ClientConfig{
		AppName:             lib.DelegatorApp + cfg.AppName,
		ServerAddr:          cfg.ServerAddr,
		HomeDir:             cfg.HomeDir,
		BinDir:              cfg.BinDir,
		NodeBaseDir:         getNodeBaseDir(cfg.HomeDir, nodeInfo.NodeID),
		LogDir:              common.NodeAgentLogPath,
		AppDataDir:          common.NodeAgentAppDataPath,
		NodeInfo:            nodeInfo,
		DeviceID:            cfg.DeviceID,
		Initialized:         initialized,
		RestartOnNodeExited: cfg.RestartOnNodeExited,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	errCh := nodeClient.SetupClient.SubscribeCommand()

	needsReconnect := false

	for {
		select {
		case <-ticker.C:
			//report node status every 5 seconds
			if err := nodeClient.StatusClient.ReportNodeStatus(); isConnectionDead(err) {
				log.Println("Error in ReportNodeStatus:", err)
				needsReconnect = true
			}

			if err := nodeClient.StatusClient.ReportStorageStatus(); isConnectionDead(err) {
				log.Println("Error in ReportNodeStorage:", err)
				needsReconnect = true
			}

		case err := <-errCh:
			//subscription error happened
			if err != nil {
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
