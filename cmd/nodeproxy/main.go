package main

import (
	"chain/internal/nodeproxy"
	cmd "chain/winservice"
	"context"
	"fmt"
	"os"
)

// Service functions
func serviceMain(ctx context.Context) error {
	fmt.Println("Service started")
	AppMain(ctx)
	return nil
}

func serviceStop() error {
	fmt.Println("Service cleanup completed")
	return nil
}

// Console function
func consoleMain(ctx context.Context) error {
	fmt.Println("Console mode")
	fmt.Println("Press Ctrl+C to exit")

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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func AppMain(ctx context.Context) {
	err := nodeproxy.StartClient()
	if err != nil {
		fmt.Println("start NodeProxy client error:", err)
	}
}
