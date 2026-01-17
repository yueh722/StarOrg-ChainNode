package main

import (
	"chain/internal/nodeproxy"
	log "chain/lib/zlog"
	"chain/utils"
	cmd "chain/winservice"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

// nodeproxy.exe -c your-path-to-config.ini
//
//	option1: load config file from the path you specified

// nodeproxy.exe
//
//	option1: load config file from %ProgramData%/chain/nodeproxy/config.ini

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
		cmd.WithName("NodeProxyService"),
		cmd.WithDisplayName("Node Proxy Service"),
	)

	// Execute the service command
	if err := cmd.NewServiceCmd().Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func AppMain(ctx context.Context, runtime cmd.ServiceRuntime) {

	appDataPath := utils.GetAppDataPath("nodeproxy")
	cfgFiles := []string{runtime.ConfigPath, filepath.Join(appDataPath, "config.ini")}

	cfg, err := nodeproxy.LoadConfig(cfgFiles, nodeproxy.GetDefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	cfgByte, _ := json.MarshalIndent(cfg, "", "  ")
	log.Info("applied config data: \n", string(cfgByte))

	err = nodeproxy.StartClient(cfg)
	if err != nil {
		log.Error("start NodeProxy client error:", err)
	} else {
		log.Info("NodeProxy client exit without error")
	}
}
