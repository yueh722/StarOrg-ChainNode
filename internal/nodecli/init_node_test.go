package nodecli

import (
	"chain/lib"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitNodeFiles(t *testing.T) {

	peers := []lib.Peer{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.1",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node02",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.2",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.3",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.4",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
	}

	peersByte, _ := json.Marshal(peers)

	genesisTime := time.Now().UnixNano()

	for _, peer := range peers {
		nodeInitArgs = &NodeInitializationArguments{
			chainId:     "chain-abc",
			genesisTime: genesisTime,
			nodeId:      peer.NodeId,
			peers:       string(peersByte),
		}

		err := initnodeFiles(nil, []string{})
		if err != nil {
			assert.Error(t, err)
		}
		fmt.Printf("initnodeFile for peer %v, %v\n", peer.NodeName, err)
	}
}

/*
func TestAddNodeFiles(t *testing.T) {

	peers := lib.PeerList{
		{
			NodeName:       "node01",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.1",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node02",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.2",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.3",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.4",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node05",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.5",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    true,
		},
		{
			NodeName:       "node06",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeAddress:    "192.168.1.6",
			P2PPort:        26656,
			RPCPort:        26657,
			ABCIPort:       26658,
			WebServicePort: 8000,
			IsValidator:    false,
		},
	}

	peersByte, _ := json.Marshal(peers)

	//load genesisDoc from file
	genesisDoc, err := types.GenesisDocFromFile("./node01/config/genesis.json")
	if err != nil {
		assert.Error(t, err)
		return
	}

	genDocBytes, err := cmtjson.Marshal(genesisDoc)
	if err != nil {
		assert.Error(t, err)
	}

	nodeAddArgs = &NodeAddArguments{
		NodeId:     peers.GetPeerByNodeName("node06").NodeId,
		Peers:      string(peersByte),
		GenesisDoc: string(genDocBytes),
	}

	err = addnodeFiles(nil, []string{})
	if err != nil {
		assert.Error(t, err)
	}

	fmt.Printf("addnodeFile for peer %v, %v\n", "node05", err)

}

*/

// 测试ConvertJsonToPeer函数
func TestConvertJsonToPeer(t *testing.T) {
	jsonStr := `[{
		"node_name": "peer123",
		"node_id": "id123",
		"priv_key": "privatekey123",
		"node_address": "192.168.1.1",
		"p2p_port": 26656,
		"rpc_port": 26657,
		"abci_port": 26658,
		"webservice_port": 8000,
		"is_validator": true
	}]`

	expectedPeer := &lib.Peer{
		NodeName:       "peer123",
		NodeId:         "id123",
		PrivKey:        "privatekey123",
		NodeAddress:    "192.168.1.1",
		P2PPort:        26656,
		RPCPort:        26657,
		ABCIPort:       26658,
		WebServicePort: 8000,
		IsValidator:    true,
	}

	peers := lib.ConvertJsonToPeers(jsonStr)

	actualPeer := peers[0]

	// 检查每个字段是否与预期值匹配
	if actualPeer.NodeName != expectedPeer.NodeName {
		t.Errorf("NodeName字段不匹配，预期值: %v, 实际值: %v", expectedPeer.NodeName, actualPeer.NodeName)
	}
	if actualPeer.NodeId != expectedPeer.NodeId {
		t.Errorf("NodeId字段不匹配，预期值: %v, 实际值: %v", expectedPeer.NodeId, actualPeer.NodeId)
	}
	if actualPeer.PrivKey != expectedPeer.PrivKey {
		t.Errorf("PrivKey字段不匹配，预期值: %v, 实际值: %v", expectedPeer.PrivKey, actualPeer.PrivKey)
	}
	if actualPeer.NodeAddress != expectedPeer.NodeAddress {
		t.Errorf("NodeAddress字段不匹配，预期值: %v, 实际值: %v", expectedPeer.NodeAddress, actualPeer.NodeAddress)
	}
	if actualPeer.P2PPort != expectedPeer.P2PPort {
		t.Errorf("P2PPort字段不匹配，预期值: %v, 实际值: %v", expectedPeer.P2PPort, actualPeer.P2PPort)
	}
	if actualPeer.RPCPort != expectedPeer.RPCPort {
		t.Errorf("RPCPort字段不匹配，预期值: %v, 实际值: %v", expectedPeer.RPCPort, actualPeer.RPCPort)
	}
	if actualPeer.ABCIPort != expectedPeer.ABCIPort {
		t.Errorf("ABCIPort字段不匹配，预期值: %v, 实际值: %v", expectedPeer.ABCIPort, actualPeer.ABCIPort)
	}
	if actualPeer.WebServicePort != expectedPeer.WebServicePort {
		t.Errorf("WebServicePort字段不匹配，预期值: %v, 实际值: %v", expectedPeer.WebServicePort, actualPeer.WebServicePort)
	}
	if actualPeer.IsValidator != expectedPeer.IsValidator {
		t.Errorf("IsValidator字段不匹配，预期值: %v, 实际值: %v", expectedPeer.IsValidator, actualPeer.IsValidator)
	}
}
