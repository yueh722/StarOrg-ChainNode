package client

import (
	"chain/lib"
	"chain/rpc/common/namespace"
	rpc2 "chain/rpc/common/response"
	types "chain/rpc/common/setupTypes"
	"chain/rpc/server"
	"chain/utils"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/samber/lo"
)

type ArgNodeList []lib.ArgNodeInfo

func init() {
	// create config directories
	utils.TryCreateDir(utils.GetAppDataPath("NodeAgent"))
	utils.TryCreateDir(utils.GetAppDataPath("NodeMaster"))
}

func MakeLocalPeerList4() ArgNodeList {
	peers := []lib.ArgNodeInfo{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 1000,
			IsValidator:    true,
		},
		{
			NodeName:       "node02",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        36656,
			RPCPort:        36657,
			ABCIPort:       36658,
			WebServicePort: 2000,
			IsValidator:    true,
		},
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        46656,
			RPCPort:        46657,
			ABCIPort:       46658,
			WebServicePort: 3000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
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
		"BroadcastPurgeChainNode",
	}

	slice.ForEach(functions, func(index int, name string) {
		methods[name] = namespace.MakeMethodName(namespace.SetupNamespace{}.GetNamesapceString(), name)
	})
}

func MakeLocalNet_Peer_01_02() ArgNodeList {
	peers := ArgNodeList{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 1000,
			IsValidator:    true,
		},
		{
			NodeName:       "node02",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        36656,
			RPCPort:        36657,
			ABCIPort:       36658,
			WebServicePort: 2000,
			IsValidator:    true,
		},
	}

	return peers
}

func MakeLocalNet_Peer_03_04() ArgNodeList {
	peers := ArgNodeList{
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        46656,
			RPCPort:        46657,
			ABCIPort:       46658,
			WebServicePort: 3000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        56656,
			RPCPort:        56657,
			ABCIPort:       56658,
			WebServicePort: 4000,
			IsValidator:    true,
		},
	}

	return peers
}

func MakeLocalNet_Peer_05_06() ArgNodeList {
	peers := ArgNodeList{
		{
			NodeName:       "node05",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        57656,
			RPCPort:        57657,
			ABCIPort:       57658,
			WebServicePort: 4700,
			IsValidator:    true,
		},
		{
			NodeName:       "node06",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        58656,
			RPCPort:        58657,
			ABCIPort:       58658,
			WebServicePort: 4800,
			IsValidator:    true,
		},
		{
			NodeName:       "node07",
			NodeId:         lib.NewNodeId(),
			NodeDomainOrIP: "127.0.0.1",
			P2PPort:        59656,
			RPCPort:        59657,
			ABCIPort:       59658,
			WebServicePort: 4900,
			IsValidator:    true,
		},
	}

	return peers
}

func getNodeIDsFromArgNodeListByName(args ArgNodeList) []string {
	return lo.Map(args, func(info lib.ArgNodeInfo, _ int) string {
		return info.NodeId
	})
}

var myclient *rpc.Client

func getConnection(t *testing.T) (*rpc.Client, error) {
	if myclient != nil {
		return myclient, nil
	}

	var err error
	wsurl := "wss://localhost:8080/websocket"
	myclient, err = Connect(&wsurl, MakeRpcHeader("simulator"))
	if err != nil {
		t.Fatal("Failed to connect:", err)
	}

	fmt.Println("✅ Connected to WebSocket server")
	return myclient, nil
}

// count:
//
//	-1 	init all nodes
//	N 	init N nodes
func initializeNodes(t *testing.T, argNodes ArgNodeList) {
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

	stateBytes, _ := json.MarshalIndent(state, "", "  ")
	log.Println(string(stateBytes))

	//update nodeID to testData
	for i, status := range state.Nodes {
		if i >= len(argNodes) {
			break
		}

		argNodes[i].NodeId = status.NodeID
		argNodes[i].NodeName = status.NodeName
		argNodes[i].NodeDomainOrIP = status.NodeName

		localIp, err := utils.FindIPByPrefix(status.IpAddrs, "192.168.1")
		if err != nil {
			localIp = strings.Join(status.IpAddrs, ",")
		}
		t.Log("node info: ", status.NodeName, localIp)
	}

	var resp rpc2.RpcResponse

	err = myclient.Call(&resp, methods["BroadcastInitializeChainNode"], "greedisgood", argNodes)
	if err != nil {
		t.Fatal("Failed to broadcast init chain node:", err)
	}
}

func Test_InitializeChainNode(t *testing.T) {

	//build a peer list that contains node01, node02.
	peers := MakeLocalNet_Peer_01_02()

	//append node03, node04 into above list.
	peers = append(peers, MakeLocalNet_Peer_03_04()...)

	initializeNodes(t, peers)
}

func getPeerInternalByID(csInternal *server.ChainStateInternal, nodeID string) *lib.Peer {
	for _, peer := range csInternal.Peers {
		if peer.NodeId == nodeID {
			return &peer
		}
	}
	return nil
}

func addNodes(t *testing.T, argNodes ArgNodeList) {
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

	stateBytes, _ := json.MarshalIndent(state, "", "  ")
	log.Println(string(stateBytes))

	assignedIndex := 0

	//update nodeID to testData
	for _, status := range state.Nodes {
		if status.Initialized {
			continue
		}

		if assignedIndex >= len(argNodes) {
			break
		}

		argNodes[assignedIndex].NodeId = status.NodeID
		argNodes[assignedIndex].NodeName = status.NodeName
		argNodes[assignedIndex].NodeDomainOrIP = status.NodeName

		t.Logf("assigned node for node %v : %v, total: %v\n", argNodes[assignedIndex].NodeId, argNodes[assignedIndex].NodeName, assignedIndex)
		assignedIndex++
	}

	// add node
	for i, node := range argNodes {
		if i >= assignedIndex {
			break
		}

		t.Log("adding node: ", node.NodeId, node.NodeName, "count", i)
		var errResult rpc2.RpcResponse
		err = myclient.Call(&errResult, methods["BroadcastAddChainNode"], node)
		if err != nil {
			t.Fatal("Failed to broadcast add chain node:", node.NodeName, err)
		}
	}

}

func Test_AddChainNode(t *testing.T) {
	addNodes(t, MakeLocalNet_Peer_03_04())
}

func getChainState(myclient *rpc.Client) (*server.ChainState, error) {
	state := &server.ChainState{}

	if err := myclient.Call(state, methods["GetChainState"]); err != nil {
		return nil, err
	}
	return state, nil
}

func SetupService_ControlNode(t *testing.T, action types.NodeAction) {
	myclient, err := getConnection(t)
	if err != nil {
		return
	}

	ctlNodes := []types.SetupControlNode{}

	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}

	//update nodeID to testData
	for _, status := range state.Nodes {
		ctlNodes = append(ctlNodes, types.SetupControlNode{
			NodeID: status.NodeID,
			Action: action,
		})
	}

	if err := myclient.Call(nil, methods["ControlNode"], ctlNodes); err != nil {
		t.Fatal("Failed to call:", err)
	}

}

func getActivatedStatusNodesByName(t *testing.T, nodeName string) []types.StatusNode {
	myclient, err := getConnection(t)
	if err != nil {
		t.Fatal(err)
	}

	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}

	ss := lo.Filter(state.Nodes, func(status types.StatusNode, _ int) bool {
		return strings.Contains(status.NodeName, nodeName)
	})

	return ss
}

func controlNodes(t *testing.T, nodeIDs []string, action types.NodeAction) {

	myclient, err := getConnection(t)
	if err != nil {
		return
	}

	var ctlNodes []types.SetupControlNode

	for _, nodeID := range nodeIDs {
		ctlNodes = append(ctlNodes, types.SetupControlNode{
			NodeID: nodeID,
			Action: action,
		})
	}

	if err := myclient.Call(nil, methods["ControlNode"], ctlNodes); err != nil {
		if action == types.NodeStart && IsAlreadyRunningError(err) {
			return
		}
		t.Fatal("Failed to call:", err)
	}
}

func Test_StartAllNodes(t *testing.T) {
	SetupService_ControlNode(t, types.NodeStart)
}

func Test_RestartAllNodes(t *testing.T) {
	SetupService_ControlNode(t, types.NodeRestart)
}

func Test_StopAllNodes(t *testing.T) {
	SetupService_ControlNode(t, types.NodeStop)
}

func Test_PurgeNode01(t *testing.T) {

	nodeIDs := lo.Map(getActivatedStatusNodesByName(t, "1"), func(status types.StatusNode, _ int) string {
		return status.NodeID
	})

	controlNodes(t, nodeIDs, types.NodePurge)
}

func Test_ResetNode01(t *testing.T) {

	nodeIDs := lo.Map(getActivatedStatusNodesByName(t, "1"), func(status types.StatusNode, _ int) string {
		return status.NodeID
	})

	controlNodes(t, nodeIDs, types.NodeReset)
}

func Test_ResetNode02(t *testing.T) {
	nodeIDs := lo.Map(getActivatedStatusNodesByName(t, "2"), func(status types.StatusNode, _ int) string {
		return status.NodeID
	})

	controlNodes(t, nodeIDs, types.NodeReset)
}

func Test_PurgeChainNode(t *testing.T) {
	myclient, err := getConnection(t)
	if err != nil {
		return
	}
	// CALL RPC METHOD

	state, err := getChainState(myclient)
	if err != nil {
		t.Fatal(err)
	}

	if state.InitializedTime == 0 {
		t.Log("chain has not initialized, no need to reset")
		return
	}

	var errResult rpc2.RpcResponse

	err = myclient.Call(&errResult, methods["BroadcastPurgeChainNode"], types.SetupPurgeChain{})
	if err != nil {
		t.Fatal("Failed to broadcast reset chain node:", err)
	}
}

func sleepSec(s int32) {
	time.Sleep(time.Duration(s) * time.Second)
}

func TestAutoDeployFromScratch(t *testing.T) {

	t.Log("start reseting chain...")
	Test_PurgeChainNode(t)

	sleepSec(10)

	t.Log("start initializing chain nodes 01, 02...")
	peers1 := MakeLocalNet_Peer_01_02()
	//peers1 = append(peers1, MakeLocalNet_Peer_03_04()...)
	initializeNodes(t, peers1)
	sleepSec(10)

	t.Log("start running nodes 01, 02...")
	controlNodes(t, getNodeIDsFromArgNodeListByName(peers1), types.NodeStart)
	sleepSec(30)

	t.Log("start adding nodes 03, 04...")
	peers2 := MakeLocalNet_Peer_03_04()
	addNodes(t, peers2)
	sleepSec(10)

	t.Log("start running nodes 03, 04...")
	controlNodes(t, getNodeIDsFromArgNodeListByName(peers2), types.NodeStart)

	t.Log("purge nodes 01")
	Test_PurgeNode01(t)

	t.Log("fetch live state and print it...")
	PrintState(t)
	//Test_RestartAllNodes(t)

	t.Log("start adding nodes 05...")
	peers5 := MakeLocalNet_Peer_05_06()
	addNodes(t, peers5)
	sleepSec(10)
	controlNodes(t, getNodeIDsFromArgNodeListByName(peers5), types.NodeStart)

	t.Log("fetch live state and print it...")
	PrintState(t)

	t.Log("restart all nodes")
	Test_StartAllNodes(t)
}

func PrintState(t *testing.T) {
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

	stateBytes, _ := json.MarshalIndent(state, "", "  ")
	log.Println(string(stateBytes))
}

func TestAddValidator(t *testing.T) {
	t.Log("start adding nodes 05, 06...")
	peers2 := MakeLocalNet_Peer_03_04()
	addNodes(t, peers2)
	sleepSec(10)
}

func TestValidPeers(t *testing.T) {
	peers := lib.ArgNodeArrayToPeers(MakeLocalNet_Peer_03_04())

	if err := lib.ValidNodeArray(peers); err != nil {
		t.Fatal(err)
	}
}

func IsAlreadyRunningError(err error) bool {
	return strings.Contains(err.Error(), "node or app already running")
}
