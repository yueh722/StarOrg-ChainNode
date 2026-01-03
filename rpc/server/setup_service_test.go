package server

import (
	"chain/lib"
	"chain/rpc/client"
	"chain/rpc/common"
	"fmt"
	"log"
	"testing"
)

func MakePeerList() lib.PeerList {
	peers := []lib.Peer{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.1",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 1000,
			IsValidator:    true,
		},
		{
			NodeName:       "node02",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.2",
			P2PPort:        36656,
			RPCPort:        36657,
			ABCIPort:       36658,
			WebServicePort: 2000,
			IsValidator:    true,
		},
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.3",
			P2PPort:        46656,
			RPCPort:        46657,
			ABCIPort:       46658,
			WebServicePort: 3000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.4",
			P2PPort:        56656,
			RPCPort:        56657,
			ABCIPort:       56658,
			WebServicePort: 4000,
			IsValidator:    true,
		},
	}

	return peers
}

func TestSetupService_BroadcastInitializeChainNode(t *testing.T) {

	wsurl := "wss://localhost:8080/ws"
	myclient, err := client.Connect(&wsurl)
	if err != nil {
		t.Fatal("Failed to connect:", err)
	}
	defer myclient.Close()

	fmt.Println("✅ Connected to WebSocket server")

	// CALL RPC METHOD

	//get chain state
	state := &ChainState{}

	method := common.MakeMethodName(common.SetupNamespace{}.GetNamesapceString(), "GetChainState")
	if err := myclient.Call(state, method); err != nil {
		t.Fatal("Failed to call:", err)
	}

	log.Println(state)

	//peer list
	peers := MakePeerList()

	//update nodeID to testData
	for i, status := range state.Nodes {
		peers[i].NodeId = status.NodeID
	}

	var errResult error
	method = common.MakeMethodName(common.SetupNamespace{}.GetNamesapceString(), "BroadcastInitializeChainNode")

	err = myclient.Call(&errResult, method, "greedisgood", peers.ToJson())
	if err != nil {
		t.Fatal("Failed to broadcast init chain node:", err)
	}
}
