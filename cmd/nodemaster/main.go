package main

import (
	"chain/internal/nodemaster"
	log "chain/lib/zlog"
	"chain/rpc/security"
	"chain/utils"
	cmd "chain/winservice"
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"time"
)

// nodemaster.exe -c your-path-to-config.ini
//
//	option1: load config file from the path you specified

// nodemaster.exe
//
//	option1: load config file from %ProgramData%/chain/nodemaster/config.ini

// Service functions
func serviceMain(ctx context.Context, runtime cmd.ServiceRuntime) error {
	log.Println("Service started")
	AppMain(ctx, runtime)
	return nil
}

func serviceStop() error {
	log.Println("Service cleanup completed")
	return nil
}

// Console function
func consoleMain(ctx context.Context, runtime cmd.ServiceRuntime) error {
	log.Println("Console mode")
	log.Println("Press Ctrl+C to exit")

	AppMain(ctx, runtime)
	return nil
}

func main() {
	// Register service functions
	cmd.RegisterService(serviceMain, serviceStop)

	// Register console function
	cmd.RegisterConsole(consoleMain)

	// Optional: set service configuration
	cmd.SetConfig(
		cmd.WithName("NodeMasterService"),
		cmd.WithDisplayName("Node master Service"),
	)

	// Execute the service command
	if err := cmd.NewServiceCmd().Execute(); err != nil {
		log.Println(err)
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

func StartMonitorConfig(cfgFile string) {
	MonitorConfig(cfgFile, func() {
		cfg, _, err := nodemaster.LoadConfig([]string{cfgFile}, nodemaster.GetDefaultConfig())
		if err != nil {
			log.Info("[reloading config] failed to load config: ", err)
			return
		}

		cfgByte, _ := json.MarshalIndent(cfg, "", "  ")
		log.Info("[reloading config] new config loaded: \n", string(cfgByte))
		nodemaster.UpdateFirewallPolicy(security.FirewallPolicy{
			FWMode:         cfg.FWMode,
			AllowOnlyHosts: cfg.AllowOnlyHosts,
		})
	})
}

func AppMain(ctx context.Context, runtime cmd.ServiceRuntime) {
	appDataPath := utils.GetAppDataPath("nodemaster")
	cfgFiles := []string{runtime.ConfigPath, filepath.Join(appDataPath, "config.ini")}

	cfg, cfgFile, err := nodemaster.LoadConfig(cfgFiles, nodemaster.GetDefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	cfgByte, _ := json.MarshalIndent(cfg, "", "  ")
	log.Info("applied config data of master: \n", string(cfgByte))

	go StartMonitorConfig(cfgFile)

	err = nodemaster.StartServer(cfg)
	if err != nil {
		log.Println("start NodeMaster server error:", err)
	} else {
		log.Println("NodeMaster server exit without error")
	}
}
