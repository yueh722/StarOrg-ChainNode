package server

import (
	"chain/lib"
	"chain/rpc/client"
	"chain/rpc/common"
	"fmt"
	"log"
	"testing"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/ethereum/go-ethereum/rpc"
)

func MakePeerList4() lib.PeerList {
	peers := []lib.Peer{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
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
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
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
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
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
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
			P2PPort:        56656,
			RPCPort:        56657,
			ABCIPort:       56658,
			WebServicePort: 4000,
			IsValidator:    true,
		},
	}

	return peers
}

var methods = map[string]string{}

func init() {

	functions := []string{
		"GetChainState",
		"BroadcastInitializeChainNode",
		"BroadcastAddChainNode",
		"ControlNode",
		"BroadcastResetChainNode",
	}

	slice.ForEach(functions, func(index int, name string) {
		methods[name] = common.MakeMethodName(common.SetupNamespace{}.GetNamesapceString(), name)
	})
}

func MakePeer_01_02() lib.PeerList {
	peers := []lib.Peer{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
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
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
			P2PPort:        36656,
			RPCPort:        36657,
			ABCIPort:       36658,
			WebServicePort: 2000,
			IsValidator:    true,
		},
	}

	return peers
}

func MakePeer_03_04() lib.PeerList {
	peers := []lib.Peer{
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
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
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
			P2PPort:        56656,
			RPCPort:        56657,
			ABCIPort:       56658,
			WebServicePort: 4000,
			IsValidator:    true,
		},
	}

	return peers
}

func getConnection(t *testing.T) (*rpc.Client, error) {
	wsurl := "wss://localhost:8080/ws"
	myclient, err := client.Connect(&wsurl)
	if err != nil {
		t.Fatal("Failed to connect:", err)
	}

	fmt.Println("✅ Connected to WebSocket server")
	return myclient, nil
}

func TestSetupService_BroadcastInitializeChainNode(t *testing.T) {

	myclient, err := getConnection(t)
	if err != nil {
		return
	}
	// CALL RPC METHOD

	//get chain state
	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(state)

	//build a peer list that contains node01, node02.
	peers := MakePeer_01_02()

	//append node03, node04 into above list.
	peers = append(peers, MakePeer_03_04()...)

	//update nodeID to testData
	for i, status := range state.Nodes {
		if i >= len(peers) {
			break
		}
		peers[i].NodeId = status.NodeID
	}

	var errResult error

	err = myclient.Call(&errResult, methods["BroadcastInitializeChainNode"], "greedisgood", peers.ToJson())
	if err != nil {
		t.Fatal("Failed to broadcast init chain node:", err)
	}
}

func getPeerInternalByID(csInternal *ChainStateInternal, nodeID string) *lib.Peer {
	for _, peer := range csInternal.Peers {
		if peer.NodeId == nodeID {
			return &peer
		}
	}
	return nil
}

func TestSetupService_BroadcastAddChainNode(t *testing.T) {

	myclient, err := getConnection(t)
	if err != nil {
		return
	}
	// CALL RPC METHOD

	//get chain state
	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(state)

	//peers will be added for chain
	peers := MakePeer_03_04()
	assignedIndex := 0

	//update nodeID to testData
	for _, status := range state.Nodes {
		if status.Initialized {
			continue
		}

		if assignedIndex >= len(peers) {
			t.Fatal("no more available peers to be added")
		}

		peers[assignedIndex].NodeId = status.NodeID
		t.Logf("assigned peer for node %v : %v, total: %v\n", peers[assignedIndex].NodeId, peers[assignedIndex].NodeName, assignedIndex)
		assignedIndex++
	}

	for _, node := range state.Nodes {
		if node.Initialized {
			continue
		}
		t.Log("uninitialized peers: ", node)
	}

	// add node
	for _, peer := range peers {

		t.Log("adding peer: ", peer.NodeId, peer.NodeName)
		var errResult error
		err = myclient.Call(&errResult, methods["BroadcastAddChainNode"], peer)
		if err != nil {
			t.Fatal("Failed to broadcast add chain node:", peer.NodeName, err)
		}
	}

}

func getChainState(myclient *rpc.Client) (*ChainState, error) {
	state := &ChainState{}

	if err := myclient.Call(state, methods["GetChainState"]); err != nil {
		return nil, err
	}
	return state, nil
}

func SetupService_ControlNode(t *testing.T, action common.NodeAction) {
	myclient, err := getConnection(t)
	if err != nil {
		return
	}

	ctlNodes := []common.SetupControlNode{}

	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}

	//update nodeID to testData
	for _, status := range state.Nodes {
		ctlNodes = append(ctlNodes, common.SetupControlNode{
			NodeID: status.NodeID,
			Action: action,
		})
	}

	if err := myclient.Call(nil, methods["ControlNode"], ctlNodes); err != nil {
		t.Fatal("Failed to call:", err)
	}

}

func TestSetupService_StartNode(t *testing.T) {
	SetupService_ControlNode(t, common.NodeStart)
}

func TestSetupService_RestartNode(t *testing.T) {
	SetupService_ControlNode(t, common.NodeRestart)
}

func TestSetupService_StopNode(t *testing.T) {
	SetupService_ControlNode(t, common.NodeStop)
}

func TestSetupService_BroadcastResetChainNode(t *testing.T) {
	myclient, err := getConnection(t)
	if err != nil {
		return
	}
	// CALL RPC METHOD

	var errResult error

	err = myclient.Call(&errResult, methods["BroadcastResetChainNode"], common.SetupResetChain{})
	if err != nil {
		t.Fatal("Failed to broadcast reset chain node:", err)
	}
}
