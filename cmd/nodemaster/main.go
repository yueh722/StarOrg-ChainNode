package main

import (
	"chain/internal/nodemaster"
	cmd "chain/winservice"
	"context"
	"log"
	"os"
)

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
		cmd.WithName("NodeMasterService"),
		cmd.WithDisplayName("Node master Service"),
	)

	// Execute the service command
	if err := cmd.NewServiceCmd().Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func AppMain(ctx context.Context) {
	err := nodemaster.StartServer()
	if err != nil {
		log.Println("start NodeMaster server error:", err)
	}
}
