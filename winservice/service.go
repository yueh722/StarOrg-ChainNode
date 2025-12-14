// winservice/service.go
package winservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	winsvc "github.com/kardianos/service"
)

// 类型定义
type MainFunc func(ctx context.Context) error
type StopFunc func() error

// Config 服务配置
type Config struct {
	Name        string   // 服务名称
	DisplayName string   // 显示名称
	Description string   // 描述
	RunAs       string   // 运行账户
	Args        []string // 服务参数
}

// Option 配置选项函数
type Option func(*Config)

// Manager 服务管理器
type Manager struct {
	config  Config
	service winsvc.Service
}

// 服务包装器，实现 winsvc.Interface
type serviceWrapper struct {
	mainFunc MainFunc
	stopFunc StopFunc
	cancel   context.CancelFunc
}

func (sw *serviceWrapper) Start(s winsvc.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	sw.cancel = cancel

	go func() {
		if sw.mainFunc != nil {
			_ = sw.mainFunc(ctx)
		}
	}()

	return nil
}

func (sw *serviceWrapper) Stop(s winsvc.Service) error {
	if sw.cancel != nil {
		sw.cancel()
	}

	if sw.stopFunc != nil {
		return sw.stopFunc()
	}

	return nil
}

// Run 运行Windows服务
// 这是给服务主程序调用的函数
func Run(mainFunc MainFunc, stopFunc StopFunc) error {
	return RunWithConfig(mainFunc, stopFunc, Config{})
}

// RunWithConfig 使用配置运行服务
func RunWithConfig(mainFunc MainFunc, stopFunc StopFunc, config Config) error {
	// 确保有服务名称
	if config.Name == "" {
		config.Name = getDefaultServiceName()
	}
	if config.DisplayName == "" {
		config.DisplayName = config.Name
	}
	if config.Description == "" {
		config.Description = config.Name
	}

	// 创建服务包装器
	wrapper := &serviceWrapper{
		mainFunc: mainFunc,
		stopFunc: stopFunc,
	}

	// 创建服务配置
	svcConfig := &winsvc.Config{
		Name:        config.Name,
		DisplayName: config.DisplayName,
		Description: config.Description,
		Arguments:   config.Args,
	}

	if config.RunAs != "" {
		svcConfig.UserName = config.RunAs
	}

	// 创建服务实例
	s, err := winsvc.New(wrapper, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// 运行服务
	return s.Run()
}

// NewManager 创建服务管理器
// 这是给安装服务的主程序调用的函数
func NewManager(opts ...Option) (*Manager, error) {
	config := Config{
		Name: getDefaultServiceName(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(&config)
	}

	// 确保有显示名称和描述
	if config.DisplayName == "" {
		config.DisplayName = config.Name
	}
	if config.Description == "" {
		config.Description = config.Name
	}

	// 创建服务包装器（空实现，仅用于管理）
	wrapper := &serviceWrapper{}

	// 创建服务配置
	svcConfig := &winsvc.Config{
		Name:        config.Name,
		DisplayName: config.DisplayName,
		Description: config.Description,
		Arguments:   config.Args,
	}

	if config.RunAs != "" {
		svcConfig.UserName = config.RunAs
	}

	// 创建服务实例
	s, err := winsvc.New(wrapper, svcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &Manager{
		config:  config,
		service: s,
	}, nil
}

// Install 安装服务
func (m *Manager) Install() error {
	return m.service.Install()
}

// Uninstall 卸载服务
func (m *Manager) Uninstall() error {
	return m.service.Uninstall()
}

// Start 启动服务
func (m *Manager) Start() error {
	return winsvc.Control(m.service, "start")
}

// Stop 停止服务
func (m *Manager) Stop() error {
	return winsvc.Control(m.service, "stop")
}

// Restart 重启服务
func (m *Manager) Restart() error {
	return winsvc.Control(m.service, "restart")
}

// Status 获取服务状态
func (m *Manager) Status() (string, error) {
	status, err := m.service.Status()
	if err != nil {
		return "", err
	}
	return strconv.Itoa(int(status)), nil
}

// WithName 设置服务名称
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithDisplayName 设置显示名称
func WithDisplayName(name string) Option {
	return func(c *Config) {
		c.DisplayName = name
	}
}

// WithDescription 设置描述
func WithDescription(desc string) Option {
	return func(c *Config) {
		c.Description = desc
	}
}

// WithRunAs 设置运行账户
func WithRunAs(user string) Option {
	return func(c *Config) {
		c.RunAs = user
	}
}

// WithArgs 设置服务参数
func WithArgs(args []string) Option {
	return func(c *Config) {
		c.Args = args
	}
}

// 获取默认服务名称
func getDefaultServiceName() string {
	exePath := os.Args[0]
	exeName := filepath.Base(exePath)
	ext := filepath.Ext(exeName)
	return strings.TrimSuffix(exeName, ext)
}
