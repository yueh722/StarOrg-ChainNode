package winservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var globalConfigPath string

// Global variables for application registration
var (
	serviceMainFunc MainFunc
	serviceStopFunc StopFunc
	consoleMainFunc func(ctx context.Context, runtime ServiceRuntime) error
	serviceOptions  []Option
)

// RegisterService registers service functions
func RegisterService(mainFunc MainFunc, stopFunc StopFunc) {
	serviceMainFunc = mainFunc
	serviceStopFunc = stopFunc
}

// RegisterConsole registers console function
func RegisterConsole(mainFunc func(ctx context.Context, runtime ServiceRuntime) error) {
	consoleMainFunc = mainFunc
}

// SetConfig sets service configuration options
func SetConfig(opts ...Option) {
	serviceOptions = opts
}

func rootCmd() *cobra.Command {
	executablePath, _ := os.Executable()
	executableName := filepath.Base(executablePath)
	cmd := &cobra.Command{
		Use: executableName,
		Run: runConsoleMode,
	}

	cmd.PersistentFlags().StringVarP(
		&globalConfigPath,
		"config",
		"c",
		"",
		"config file path (ini file)",
	)

	return cmd
}

// NewServiceCmd creates the service management command
func NewServiceCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage Windows service",
		Long: `Use this command to install, uninstall, start, stop Windows service.
When executed without subcommands, runs in service mode.`,
		// Run service mode when no subcommands
		RunE: runServiceMode,
	}

	// Add service management subcommands
	cmd.AddCommand(
		newServiceInstallCmd(),
		newServiceUninstallCmd(),
		newServiceStartCmd(),
		newServiceStopCmd(),
		newServiceStatusCmd(),
	)

	rootCmd := rootCmd()
	rootCmd.AddCommand(cmd)
	return rootCmd
}

// runConsoleMode runs console mode (when no subcommands provided)
func runConsoleMode(cmd *cobra.Command, args []string) {
	if consoleMainFunc == nil {
		fmt.Fprintln(os.Stderr, "Console function not registered")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := consoleMainFunc(ctx, ServiceRuntime{ConfigPath: globalConfigPath}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newServiceInstallCmd creates service install command
func newServiceInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Windows service",
		Long: `Install Windows service.
After installation, service binary path will be: [current-exe-path] service run`,
		RunE: runServiceInstall,
	}

	// Optional configuration flags for installation
	cmd.Flags().String("display-name", "", "Display name for the service")
	cmd.Flags().String("description", "", "Description of the service")
	cmd.Flags().String("run-as", "", "Service account (e.g., NT AUTHORITY\\LocalService)")

	return cmd
}

// newServiceUninstallCmd creates service uninstall command
func newServiceUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Windows service",
		Long:  `Uninstall Windows service.`,
		RunE:  runServiceUninstall,
	}
}

// newServiceStartCmd creates service start command
func newServiceStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start Windows service",
		Long:  `Start Windows service.`,
		RunE:  runServiceStart,
	}
}

// newServiceStopCmd creates service stop command
func newServiceStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop Windows service",
		Long:  `Stop Windows service.`,
		RunE:  runServiceStop,
	}
}

// newServiceStatusCmd creates service status command
func newServiceStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check service status",
		Long:  `Check Windows service status.`,
		RunE:  runServiceStatus,
	}
}

// newServiceRunCmd creates service run command (internal use only)
func newServiceRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "run",
		Short:  "Run in service mode (internal use)",
		Hidden: true,
		RunE:   runServiceMode,
	}
}

// runServiceMode runs service mode (for "service run" command)
func runServiceMode(cmd *cobra.Command, args []string) error {
	if serviceMainFunc == nil || serviceStopFunc == nil {
		return fmt.Errorf("service functions not registered")
	}

	// Run service (using Run)
	return Run(serviceMainFunc, serviceStopFunc)
}

// runServiceInstall runs service installation
func runServiceInstall(cmd *cobra.Command, args []string) error {
	// Get command line flags
	displayName, _ := cmd.Flags().GetString("display-name")
	description, _ := cmd.Flags().GetString("description")
	runAs, _ := cmd.Flags().GetString("run-as")

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare service arguments: when service starts, execute "service run" command
	serviceArgs := []string{"service"}

	// Build service configuration options
	opts := make([]Option, 0, len(serviceOptions)+4)
	opts = append(opts, serviceOptions...)
	opts = append(opts, WithArgs(serviceArgs))

	if displayName != "" {
		opts = append(opts, WithDisplayName(displayName))
	}

	if description != "" {
		opts = append(opts, WithDescription(description))
	}

	if runAs != "" {
		opts = append(opts, WithRunAs(runAs))
	}

	// Create service manager
	manager, err := NewManager(opts...)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	// Display installation information
	fmt.Println("Installing Windows service...")
	fmt.Printf("Service Name:     %s\n", manager.GetServiceName())
	fmt.Printf("Executable:       %s\n", exePath)
	fmt.Printf("Service Args:     %s\n", serviceArgs)
	if displayName != "" {
		fmt.Printf("Display Name:     %s\n", displayName)
	}
	if description != "" {
		fmt.Printf("Description:      %s\n", description)
	}
	if runAs != "" {
		fmt.Printf("Run As Account:   %s\n", runAs)
	}
	fmt.Println()

	// Execute installation
	if err := manager.Install(); err != nil {
		return fmt.Errorf("service installation failed: %w", err)
	}

	fmt.Println("✅ Service installed successfully")
	fmt.Println("\nNext steps:")
	fmt.Println("  Start service:    app service start")
	fmt.Println("  Stop service:     app service stop")
	fmt.Println("  Check status:     app service status")
	fmt.Println("  Uninstall:        app service uninstall")

	return nil
}

// runServiceUninstall runs service uninstallation
func runServiceUninstall(cmd *cobra.Command, args []string) error {
	// Create service manager
	manager, err := NewManager(serviceOptions...)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	fmt.Printf("Uninstalling service: %s\n", manager.GetServiceName())

	if err := manager.Uninstall(); err != nil {
		return fmt.Errorf("service uninstallation failed: %w", err)
	}

	fmt.Println("✅ Service uninstalled successfully")
	return nil
}

// runServiceStart runs service start
func runServiceStart(cmd *cobra.Command, args []string) error {
	// Create service manager
	manager, err := NewManager(serviceOptions...)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	fmt.Printf("Starting service: %s\n", manager.GetServiceName())

	if err := manager.Start(); err != nil {
		return fmt.Errorf("service start failed: %w", err)
	}

	fmt.Println("✅ Service started successfully")
	return nil
}

// runServiceStop runs service stop
func runServiceStop(cmd *cobra.Command, args []string) error {
	// Create service manager
	manager, err := NewManager(serviceOptions...)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	fmt.Printf("Stopping service: %s\n", manager.GetServiceName())

	if err := manager.Stop(); err != nil {
		return fmt.Errorf("service stop failed: %w", err)
	}

	fmt.Println("✅ Service stopped successfully")
	return nil
}

// runServiceStatus runs service status check
func runServiceStatus(cmd *cobra.Command, args []string) error {
	// Create service manager
	manager, err := NewManager(serviceOptions...)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	status, err := manager.Status()
	if err != nil {
		// Service might not be installed
		return fmt.Errorf("failed to get service status (service may not be installed): %w", err)
	}

	fmt.Printf("Service Name: %s\n", manager.GetServiceName())
	fmt.Printf("Service Status: %s\n", status)

	return nil
}
