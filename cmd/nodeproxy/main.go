package main

import (
	"chain/internal/common"
	"chain/internal/nodeproxy"
	"chain/lib"
	log "chain/lib/zlog"
	"chain/utils"
	service "chain/winservice"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/duke-git/lancet/v2/slice"
)

// nodeproxy.exe -c your-path-to-config.ini
//
//	option1: load config file from the path you specified

// nodeproxy.exe
//
//	option1: load config file from %ProgramData%/chain/nodeproxy/config.ini

// Service functions
func serviceMain(ctx context.Context, runtime service.ServiceRuntime) error {
	log.Println("Service started")

	if runtime.AppName == "" {
		AppLauncher(ctx, runtime)
	} else {
		AppMain(ctx, runtime)
	}
	return nil
}

func serviceStop() error {
	log.Println("Service shutting down")
	//signal terminate
	os.Exit(1)
	return nil
}

// Console function
func consoleMain(ctx context.Context, runtime service.ServiceRuntime) error {
	log.Println("Console mode")
	log.Println("Press Ctrl+C to exit")

	if runtime.AppName == "" {
		AppLauncher(ctx, runtime)
	} else {
		AppMain(ctx, runtime)
	}
	return nil
}

func main() {
	// Register service functions
	service.RegisterService(serviceMain, serviceStop)

	// Register console function
	service.RegisterConsole(consoleMain)

	// Optional: set service configuration
	service.SetConfig(
		service.WithName("NodeAgentService"),
		service.WithDisplayName("Node Agent Service"),
	)

	// Execute the service command
	if err := service.NewServiceCmd().Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// LaunchAppInternal use go-routine to run a node agent
func LaunchAppInternal(ctx context.Context, appName string, runtime service.ServiceRuntime) {

	log.Println("Start App: ", appName)
	//won't return until finished or failure happened.
	AppMain(ctx, service.ServiceRuntime{ConfigPath: runtime.ConfigPath, AppName: appName})
	log.Println("AppMain returned: ", appName)
}

func KillNodeAppProcess(homeDir string) {
	nodeInfo, err := lib.GetNodeInfo(homeDir)
	if err != nil {
		return
	}

	//Try to kill previous node & app process if existed. Because node proxy might exit unexpected last time.
	//In order to capture log data of node and abci-app, node proxy need to kill their process and restart them.
	_ = lib.KillNode(nodeInfo.NodeID)
}

// LaunchAppExternal create process to run a node agent
func LaunchAppExternal(ctx context.Context, appCfg nodeproxy.AppConfig, runtime service.ServiceRuntime) {

	KillNodeAppProcess(appCfg.HomeDir)

	l := lib.Launcher{
		Name:        "Run-NodeAgent",
		AppFilePath: utils.GetCurrentProcessInfo().ImagePath,
		Args: map[string]string{
			"-n": appCfg.AppName,
		},
		Logger: log.NewStdoutLogger(),
	}

	if runtime.ConfigPath != "" {
		l.Args["-c"] = runtime.ConfigPath
	}

	log.Println("Create process for App: ", appCfg.AppName)

	cmd := l.NewCommandFromMap(l.AppFilePath, l.Args)
	if err := l.RunCommand(cmd); err != nil {
		log.Infof("process of app '%s' exit with error: %v", appCfg.AppName, err)
	} else {
		log.Println("process of app '%s' exit without error", appCfg.AppName)
	}
}

func AppLauncher(ctx context.Context, runtime service.ServiceRuntime) {
	cfgFiles := []string{runtime.ConfigPath, filepath.Join(common.NodeProxyAppDataPath, "config.ini")}

	if err := nodeproxy.InitializeDeviceID(cfgFiles); err != nil {
		log.Fatal("Failed to initialize device ID", err)
		return
	}

	//loading nodes from config file and launch them all with separate processes.
	appCfgs, err := nodeproxy.GetManagedApps(cfgFiles)
	if err != nil {
		log.Fatal("Get managed app names failed:", err)
		return
	}

	log.Info("loaded managed app names:", slice.Map(appCfgs, func(_ int, cfg nodeproxy.AppConfig) string {
		return cfg.AppName
	}))

	var wg = &sync.WaitGroup{}

	//AppLauncher won't return from here, start monitor all go-routine or children processes
	for _, appCfg := range appCfgs {
		wg.Add(1)

		go func() {
			defer func() {
				wg.Done()
			}()

			for {
				//LaunchAppInternal(ctx, appName, runtime)
				LaunchAppExternal(ctx, appCfg, runtime)
			}
		}()
	}

	wg.Wait()
	log.Println("Launch app finished")
}

func AppMain(ctx context.Context, runtime service.ServiceRuntime) {

	cfgFiles := []string{runtime.ConfigPath, filepath.Join(common.NodeProxyAppDataPath, "config.ini")}

	cfg, err := nodeproxy.LoadConfig(cfgFiles, runtime.AppName)
	if err != nil {
		log.Fatal(err)
	}

	cfgByte, _ := json.MarshalIndent(cfg, "", "  ")
	log.Info("applied config data: \n", string(cfgByte))

	err = nodeproxy.StartClient(ctx, cfg)
	if err != nil {
		log.Error("start NodeProxy client error:", err)
	} else {
		log.Info("NodeProxy client exit without error")
	}
}
