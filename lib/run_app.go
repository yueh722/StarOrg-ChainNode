package lib

import (
	"chain/lib/log"
	"path/filepath"
)

type RunItem struct {
	run    bool
	exitCh chan bool
	inst   *Instance
}

func RunApp(inst *Instance, exitCh chan<- bool) error {

	return nil
}

func GetAppNameByNodeID(nodeID string) string {
	return nodeID + "_app"
}

func BuildNodeLauncher(nodeID string, arg NodeArg) *Launcher {

	l := &Launcher{
		Name:        "Start-Cometbft-Node-id",
		AppFilePath: filepath.Join(arg.BinPath, "cometbft.exe"),
		Args: map[string]string{
			"node":   "",
			"--home": filepath.Join(arg.NodeBaseDir, arg.NodeInfo.NodeName),
		},
		Envs:   nil,
		Logger: log.NewCometbftNodeLogger(nodeID, arg.LogStore),
	}

	return l
}

func BuildAppLauncher(nodeID string, arg NodeArg) *Launcher {

	//load app configuration

	//get server information

	l := &Launcher{
		Name:        "Start-Cometbft-App",
		AppFilePath: filepath.Join(arg.BinPath, "abci-v2-Identity-app.exe"),
		Args: map[string]string{
			"--home":          filepath.Join(arg.NodeBaseDir, "app"),
			"--socket-addr":   arg.NodeInfo.ProxyApp,       // proxy_app
			"--http-addr":     arg.NodeInfo.WebServiceAddr, //webservice
			"--abci-rpc-addr": arg.NodeInfo.RPCAddr,        // rpc
		},
		Envs:   nil,
		Logger: log.NewCometbftNodeLogger(GetAppNameByNodeID(nodeID), arg.LogStore),
	}

	return l
}

func StartApp(nodeID string, arg NodeArg) error {
	items := []RunItem{
		{
			run:    true,
			exitCh: make(chan bool, 1),
			inst:   NewInstance(BuildNodeLauncher(nodeID, arg)),
		},
		{
			run:    true,
			exitCh: make(chan bool, 1),
			inst:   NewInstance(BuildAppLauncher(nodeID, arg)),
		},
	}

	go func() {
		for {
			for _, item := range items {
				if item.run {
					item.run = false
					go RunApp(item.inst, item.exitCh)
				}
			}

			select {
			case <-items[0].exitCh:
				items[0].run = true
			case <-items[1].exitCh:
				items[1].run = true
			}
		}
	}()

	return nil
}
