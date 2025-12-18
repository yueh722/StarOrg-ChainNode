package main

import (
	cmd "Chain/winservice"
	"context"
	"fmt"
	"os"
	"time"
)

// Service functions
func serviceMain(ctx context.Context) error {
	fmt.Println("Service started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Service stopping...")
			return nil
		case t := <-ticker.C:
			fmt.Printf("Service running at: %s\n", t.Format("2006-01-02 15:04:05"))
		}
	}
}

func serviceStop() error {
	fmt.Println("Service cleanup completed")
	return nil
}

// Console function
func consoleMain(ctx context.Context) error {
	fmt.Println("Console mode")
	fmt.Println("Press Ctrl+C to exit")

	<-ctx.Done()
	return nil
}

func main() {
	// Register service functions
	cmd.RegisterService(serviceMain, serviceStop)

	// Register console function
	cmd.RegisterConsole(serviceMain)

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
