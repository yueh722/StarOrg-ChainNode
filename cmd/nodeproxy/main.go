package main

import (
	"chain/internal/nodeproxy"
	log "chain/lib/zlog"
	cmd "chain/winservice"
	"context"
	"os"
)

// nodeproxy.exe -c your-path-to-config.ini
//
//	option1: load config file from the path you specified

// nodeproxy.exe
//
//	option1: load config file from %ProgramData%/chain/nodeproxy/config.ini
//	option2: load config file from %appdata%/chain/nodeproxy/config.ini

// Service functions
func serviceMain(ctx context.Context) error {
	log.Println("Service started")
	AppMain(ctx)
	return nil
}

func serviceStop() error {
	log.Println("Service cleanup completed")
	return nil
}

// Console function
func consoleMain(ctx context.Context) error {
	log.Println("Console mode")
	log.Println("Press Ctrl+C to exit")

	AppMain(ctx)
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

func AppMain(ctx context.Context) {
	err := nodeproxy.StartClient()
	if err != nil {
		log.Println("start NodeProxy client error:", err)
	} else {
		log.Println("NodeProxy client exit without error")
	}
}
