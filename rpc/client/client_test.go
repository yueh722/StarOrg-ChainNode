package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
)

func TestClient(t *testing.T) {

	client, err := rpc.DialContext(context.Background(), "wss://localhost:8080/ws")
	if err != nil {
		t.Fatal("Failed to Connect:", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to WebSocket server")

	//result: [nil: no result required] [pointer to any variable]
	//method: "ServiceName_methodFunctionName" ; The First letter of method function name should be lowed.
	err = client.Call(nil, "status_reportNodeStatus", nil)
	if err != nil {
		t.Fatal("Failed to call RPC method:", err)
	}
}
