package lib

import (
	"chain/lib/log"
	"fmt"
	"path/filepath"
)

type RunItem struct {
	run    bool
	exitCh chan bool
	inst   *Instance
}

var nodeRunTime NodeRuntime

func RunApp(inst *Instance, exitCh chan<- bool) error {

	defer func() {
		exitCh <- true
	}()

	if err := inst.Start(); err != nil {
		return err
	}

	code, err := inst.Wait()
	if err != nil {
		return fmt.Errorf("launcher exited with error: %v, code: %v", err, code)
	}

	return nil
}

func GetAppNameByNodeID(nodeID string) string {
	return nodeID + "_app"
}

func BuildNodeLauncher(nodeID string, arg NodeArg) *Launcher {

	l := &Launcher{
		Name:        "Start-Cometbft-Node-id",
		AppFilePath: filepath.Join(arg.BinPath, NodeFileName),
		Args: map[string]string{
			"node":   "",
			"--home": filepath.Join(arg.NodeBaseDir),
		},
		Envs:   nil,
		Logger: log.NewCometbftNodeLogger(nodeID, filepath.Join(arg.LogFilePath, "node", "node.log"), arg.LogStore),
	}

	return l
}

func BuildAppLauncher(nodeID string, arg NodeArg) *Launcher {

	l := &Launcher{
		Name:        "Start-Cometbft-App",
		AppFilePath: filepath.Join(arg.BinPath, AppFileName),
		Args: map[string]string{
			"--home":          filepath.Join(arg.NodeBaseDir, "app"),
			"--socket-addr":   arg.NodeInfo.ProxyApp,       // proxy_app
			"--http-addr":     arg.NodeInfo.WebServiceAddr, //webservice
			"--abci-rpc-addr": arg.NodeInfo.RPCAddr,        // rpc
		},
		Envs:   nil,
		Logger: log.NewCometbftNodeLogger(GetAppNameByNodeID(nodeID), filepath.Join(arg.LogFilePath, "app", "app.log"), arg.LogStore),
	}

	return l
}

func StartNodeWithArgLoop(nodeID string, arg NodeArg) NodeRuntime {
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

	return &NodeRuntimeImpl{
		NodeInst: items[0].inst,
		AppInst:  items[1].inst,
	}
}

func StartNodeWithArg(nodeID string, arg NodeArg) NodeRuntime {
	items := []RunItem{
		{
			inst: NewInstance(BuildNodeLauncher(nodeID, arg)),
		},
		{
			inst: NewInstance(BuildAppLauncher(nodeID, arg)),
		},
	}

	runtime := &NodeRuntimeImpl{
		NodeInst: items[0].inst,
		AppInst:  items[1].inst,
	}

	start_failure := false
	for _, item := range items {
		if err := item.inst.Start(); err != nil {
			start_failure = true
		}
	}

	if start_failure {
		_ = StopNode(runtime)
	} else {
		nodeRunTime = runtime
	}

	return runtime
}
