package main

import (
	"chain/internal/common"
	"chain/internal/nodeagent"
	"chain/lib"
	log "chain/lib/zlog"
	"chain/utils"
	service "chain/winservice"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/slice"
)

// nodeagent.exe -c your-path-to-config.ini
//
//	option1: load config file from the path you specified

// nodeagent.exe
//
//	option1: load config file from %ProgramData%/chainnode/nodeagent/config.ini

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
func LaunchAppExternal(ctx context.Context, appCfg nodeagent.AppConfig, runtime service.ServiceRuntime) {

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

func LaunchApp(ctx context.Context, appCfgs []nodeagent.AppConfig, runtime service.ServiceRuntime) {

	for _, appCfg := range appCfgs {

		go func() {
			for {
				//LaunchAppInternal(ctx, appCfg.AppName, runtime)
				LaunchAppExternal(ctx, appCfg, runtime)
			}
		}()
	}

}

func MonitorConfig(cfgFile string, reload func()) {
	// create watcher that will call our reload function after debounced changes
	watcher, err := utils.NewConfigWatcher(cfgFile,
		utils.WithDebounceTime(1000*time.Millisecond),
		utils.WithOnChange(reload),
		utils.WithOnError(func(err error) {
			log.Printf("watcher error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	var wg = &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func AppLauncher(ctx context.Context, runtime service.ServiceRuntime) {
	cfgFiles := []string{runtime.ConfigPath, filepath.Join(common.NodeAgentAppDataPath, "config.ini")}

	if err := nodeagent.InitializeDeviceID(cfgFiles); err != nil {
		log.Fatal("Failed to initialize device ID", err)
		return
	}

	//loading nodes from config file and launch them all with separate processes.
	appCfgs, cfgFile, err := nodeagent.GetManagedApps(cfgFiles)
	if err != nil {
		log.Fatal("Get managed app names failed:", err)
		return
	}

	log.Info("loaded managed app names:", slice.Map(appCfgs, func(_ int, cfg nodeagent.AppConfig) string {
		return cfg.AppName
	}))

	LaunchApp(ctx, appCfgs, runtime)

	MonitorConfig(cfgFile, func() {
		log.Info("[reloading config] trying to load new apps from config file...")

		reloadingAppCfgs, _, err := nodeagent.GetManagedApps([]string{cfgFile})
		if err != nil {
			log.Info("[reloading config] Get managed app names failed:", err)
			return
		}

		newAppCfgs := slice.Difference(reloadingAppCfgs, appCfgs)
		if len(newAppCfgs) > 0 {
			LaunchApp(ctx, newAppCfgs, runtime)
			log.Println("[reloading config] new app are launched, ", newAppCfgs)
			return
		}

		log.Info("[reloading config] no new apps found.")
	})

	log.Println("Launch app finished")
}

func AppMain(ctx context.Context, runtime service.ServiceRuntime) {

	cfgFiles := []string{runtime.ConfigPath, filepath.Join(common.NodeAgentAppDataPath, "config.ini")}

	cfg, err := nodeagent.LoadConfig(cfgFiles, runtime.AppName)
	if err != nil {
		log.Fatal(err)
	}

	cfgByte, _ := json.MarshalIndent(cfg, "", "  ")
	log.Info("applied config data: \n", string(cfgByte))

	err = nodeagent.StartClient(ctx, cfg)
	if err != nil {
		log.Error("start NodeProxy client error:", err)
	} else {
		log.Info("NodeProxy client exit without error")
	}
}
