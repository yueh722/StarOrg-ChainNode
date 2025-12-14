package winservice

import (
	"testing"
	"time"
)

var manager, _ = NewManager()

// TestIntegrationFullCycle 测试完整的服务生命周期
func TestIntegrationFullCycle(t *testing.T) {

	testServiceName := getDefaultServiceName()
	t.Logf("Testing with service name: %s", testServiceName)

	// 步骤1：安装服务
	t.Run("Install", func(t *testing.T) {
		err := manager.Install()
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}

	})

	// 步骤2：启动服务
	t.Run("Start", func(t *testing.T) {
		err := manager.Start()
		if err != nil {
			t.Errorf("Start failed: %v", err)
		}

		// 等待服务启动
		time.Sleep(3 * time.Second)

		_, err = manager.Status()
		if err != nil {
			t.Errorf("Failed to get status: %v", err)
		}
	})

	// 步骤3：停止服务
	t.Run("Stop", func(t *testing.T) {
		err := manager.Stop()
		if err != nil {
			t.Errorf("Stop failed: %v", err)
		}

		// 等待服务停止
		time.Sleep(2 * time.Second)

		_, err = manager.Status()
		if err != nil {
			t.Errorf("Failed to get status: %v", err)
		}
	})

	// 步骤4：卸载服务
	t.Run("Uninstall", func(t *testing.T) {
		err := manager.Uninstall()
		if err != nil {
			t.Errorf("Uninstall failed: %v", err)
		}

	})
}
