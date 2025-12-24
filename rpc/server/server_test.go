package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

func TestMathService(t *testing.T) {

	go startServer(t)

	time.Sleep(3 * time.Second)

	startClient(t)
}

func startClient(t *testing.T) {
	fmt.Println("Testing WebSocket RPC Client...")

	// 创建客户端连接

	// 注意：这里使用DialContext，但实际需要WebSocket连接
	// go-ethereum的客户端会自动检测协议

	// 方式1: 使用自定义的DialContext（推荐）
	client, err := rpc.DialContext(context.Background(), "ws://localhost:8080/ws")
	if err != nil {
		t.Fatal("Failed to connect:", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to WebSocket server")

	// 测试普通RPC调用
	var result float64
	err = client.Call(&result, "custom_add", 10.5, 20.3)
	if err != nil {
		t.Errorf("RPC call failed: %v", err)
	} else {
		t.Logf("10.5 + 20.3 = %.2f\n", result)
	}

	// 测试订阅
	fmt.Println("\nSubscribing to counter updates...")

	// 3. 订阅服务端主动推送
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counterChan := make(chan map[string]interface{}, 1000)
	// 创建订阅（通过自定义方法 custom_subscribeCustom）
	sub, err := client.Subscribe(ctx, "custom", counterChan, "subscribeStats")
	if err != nil {
		t.Fatal(fmt.Sprintf("创建订阅失败: %v", err))
	}

	fmt.Println("订阅已创建，等待服务端推送...")

	go func() {
		for {
			counter := <-counterChan
			t.Logf("counter: %v", counter)
		}
	}()

	// 4. 处理退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		fmt.Println("收到退出信号，关闭客户端...")
	case <-sub.Err():
		fmt.Printf("订阅异常: %v\n", sub.Err())
	}

	// 取消订阅
	sub.Unsubscribe()
}

func startServer(t *testing.T) {

}
