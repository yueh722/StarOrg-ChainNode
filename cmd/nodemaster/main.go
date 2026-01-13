package main

import (
	"chain/internal/nodemaster"
	log "chain/lib/zlog"
	"chain/utils"
	cmd "chain/winservice"
	"context"
	"path/filepath"
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

func AppMain(ctx context.Context, runtime cmd.ServiceRuntime) {
	appDataPath := utils.GetAppDataPath("nodemaster")
	cfgFiles := []string{runtime.ConfigPath, filepath.Join(appDataPath, "config.ini")}

	cfg, err := nodemaster.LoadConfig(cfgFiles, nodemaster.GetDefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	err = nodemaster.StartServer(cfg)
	if err != nil {
		log.Println("start NodeMaster server error:", err)
	} else {
		log.Println("NodeMaster server exit without error")
	}
}
