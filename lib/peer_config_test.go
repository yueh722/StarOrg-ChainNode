package lib

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var nodeDir, _ = filepath.Abs("./testDir")

func newNodeId() string {
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	return nodeKey.ID()
}

func newPrivKey() string {
	prvKey := ed25519.GenPrivKey()

	return base64.StdEncoding.EncodeToString(prvKey.Bytes())
}

// TestNewPeerConfig 测试NewPeerConfig函数
func TestNewPeerConfig(t *testing.T) {
	// 测试用的Peer实例
	currPeer := &Peer{
		NodeName:       "testNode",
		NodeId:         newNodeId(),
		PrivKey:        newPrivKey(),
		NodeAddress:    "127.0.0.1",
		P2PPort:        26656,
		RPCPort:        26657,
		ABCIPort:       26658,
		WebServicePort: 8080,
		IsValidator:    true,
	}

	// 正常情况
	t.Run("正常情况", func(t *testing.T) {
		persistentPeers := "peer1,peer2"

		peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
		require.NoError(t, err)
		defer os.RemoveAll(nodeDir)

		assert.NotNil(t, peerConfig.config)
		assert.Equal(t, nodeDir, peerConfig.nodeDir)
		//assert.Equal(t, currPeer.PrivKey, peerConfig.privKey)
		assert.Equal(t, currPeer.NodeId, peerConfig.nodeId)
		assert.Nil(t, peerConfig.genesis)
	})

	// 测试currPeer为nil的情况
	t.Run("currPeer为nil的情况", func(t *testing.T) {
		var currPeer *Peer
		persistentPeers := "peer1,peer2"

		peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
		require.Nil(t, peerConfig)
		assert.EqualError(t, err, "currPeer is nil")
	})

	// 测试nodeDir为空的情况
	t.Run("nodeDir为空的情况", func(t *testing.T) {
		persistentPeers := "peer1,peer2"
		var nodeDir string

		peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
		defer os.RemoveAll(peerConfig.nodeDir)

		require.NoError(t, err)
		assert.Equal(t, "./testNode", peerConfig.nodeDir)
	})

	// 测试端口超出范围的情况
	t.Run("端口超出范围的情况", func(t *testing.T) {
		currPeer.P2PPort = 65535
		persistentPeers := "peer1,peer2"

		peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
		defer os.RemoveAll(nodeDir)

		require.Nil(t, peerConfig)
		assert.EqualError(t, err, "failed to update p2p listen address: port 65536 is too large")
	})
}

// TestPeerConfig_NewGensis 测试NewGensis函数
func TestPeerConfig_NewGensis(t *testing.T) {
	currPeer := &Peer{
		NodeName:       "testNode",
		NodeId:         newNodeId(),
		PrivKey:        newPrivKey(),
		NodeAddress:    "127.0.0.1",
		P2PPort:        26656,
		RPCPort:        26657,
		ABCIPort:       26658,
		WebServicePort: 8080,
		IsValidator:    true,
	}
	persistentPeers := "peer1,peer2"

	peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
	defer os.RemoveAll(nodeDir)
	require.NoError(t, err)

	// 正常情况
	t.Run("正常情况", func(t *testing.T) {
		chainId := "testChain"
		validators := []types.GenesisValidator{
			{
				Address: ed25519.GenPrivKey().PubKey().Address(),
				PubKey:  ed25519.GenPrivKey().PubKey(),
				Power:   10,
				Name:    "validator1",
			},
		}

		err := peerConfig.NewGensis(chainId, time.Now(), validators)
		require.NoError(t, err)

		assert.NotNil(t, peerConfig.genesis)
		assert.Equal(t, "chain-testChain", peerConfig.genesis.ChainID)
		assert.NotNil(t, peerConfig.genesis.ConsensusParams)
		assert.Equal(t, 10, int(peerConfig.genesis.Validators[0].Power))
	})

	// 测试genesisTime为零时间的情况
	t.Run("genesisTime为零时间的情况", func(t *testing.T) {
		chainId := "testChain"
		validators := []types.GenesisValidator{}

		err := peerConfig.NewGensis(chainId, time.Time{}, validators)
		require.NoError(t, err)

		assert.NotNil(t, peerConfig.genesis)
		assert.Equal(t, "chain-testChain", peerConfig.genesis.ChainID)
		assert.Equal(t, time.Time{}, peerConfig.genesis.GenesisTime)
		assert.Equal(t, int64(0), peerConfig.genesis.InitialHeight)
		assert.Empty(t, peerConfig.genesis.Validators)
	})

	// 测试validators为空的情况
	t.Run("validators为空的情况", func(t *testing.T) {
		chainId := "testChain"
		validators := []types.GenesisValidator{}

		err := peerConfig.NewGensis(chainId, time.Now(), validators)
		require.NoError(t, err)

		assert.NotNil(t, peerConfig.genesis)
		assert.Equal(t, "chain-testChain", peerConfig.genesis.ChainID)
		assert.Empty(t, peerConfig.genesis.Validators)
	})
}

// TestPeerConfig_SetGenesis 测试SetGenesis函数
func TestPeerConfig_SetGenesis(t *testing.T) {
	currPeer := &Peer{
		NodeName:       "testNode",
		NodeId:         newNodeId(),
		PrivKey:        newPrivKey(),
		NodeAddress:    "127.0.0.1",
		P2PPort:        26656,
		RPCPort:        26657,
		ABCIPort:       26658,
		WebServicePort: 8080,
		IsValidator:    true,
	}
	persistentPeers := "peer1,peer2"

	peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
	defer os.RemoveAll(nodeDir)
	require.NoError(t, err)

	privKey := ed25519.GenPrivKey()
	pubKey := base64.StdEncoding.EncodeToString(privKey.PubKey().Bytes())
	address := hex.EncodeToString(privKey.PubKey().Address())

	// 测试正常情况
	t.Run("正常情况", func(t *testing.T) {
		genesisDoc := `{
			"chain_id": "chain-testChain",
			"validators": [
				{
					"address": "%s",
					"pub_key": {
						"type": "tendermint/PubKeyEd25519",
						"value": "%s"
					},
					"power": "10",
					"name": "validator1"
				}
			]
		}`

		genesisDoc = fmt.Sprintf(genesisDoc, address, pubKey)

		err := peerConfig.SetGenesis(genesisDoc, nil)
		require.NoError(t, err)

		assert.NotNil(t, peerConfig.genesis)
		assert.Equal(t, "chain-testChain", peerConfig.genesis.ChainID)
		assert.Equal(t, 1, len(peerConfig.genesis.Validators))
	})

	// 测试genesisDoc格式不正确的情况
	t.Run("genesisDoc格式不正确的情况", func(t *testing.T) {
		genesisDoc := `{
			"chain_id": "chain-testChain",
			"validators": [
				{
					"address": "%s", // address格式不正确
					"pub_key": {
						"type": "tendermint/PubKeyEd25519",
						"value": "%s"
					},
					"power": "10",
					"name": "validator1"
				}
			]
		}`

		genesisDoc = fmt.Sprintf(genesisDoc, address, pubKey)

		err := peerConfig.SetGenesis(genesisDoc, nil)
		assert.Error(t, err)
	})

	// 测试genesisDoc为空的情况
	t.Run("genesisDoc为空的情况", func(t *testing.T) {
		var genesisDoc string

		err := peerConfig.SetGenesis(genesisDoc, nil)
		assert.Error(t, err)
	})
}

// TestPeerConfig_SaveFiles 测试SaveFiles函数
func TestPeerConfig_SaveFiles(t *testing.T) {
	currPeer := &Peer{
		NodeName:       "testNode",
		NodeId:         newNodeId(),
		PrivKey:        newPrivKey(),
		NodeAddress:    "127.0.0.1",
		P2PPort:        26656,
		RPCPort:        26657,
		ABCIPort:       26658,
		WebServicePort: 8080,
		IsValidator:    true,
	}
	persistentPeers := "peer1,peer2"

	peerConfig, err := NewPeerConfig(currPeer, persistentPeers, nodeDir)
	defer os.RemoveAll(nodeDir)
	require.NoError(t, err)

	// 测试正常情况
	t.Run("正常情况", func(t *testing.T) {
		chainId := "testChain"
		validators := []types.GenesisValidator{
			{
				Address: ed25519.GenPrivKey().PubKey().Address(),
				PubKey:  ed25519.GenPrivKey().PubKey(),
				Power:   10,
				Name:    "validator1",
			},
		}

		err := peerConfig.NewGensis(chainId, time.Now(), validators)
		require.NoError(t, err)

		err = peerConfig.SaveFiles()
		require.NoError(t, err)

		// 检查文件是否存在
		_, err = os.Stat(filepath.Join(nodeDir, "config", "config.toml"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(nodeDir, "config", "priv_validator_key.json"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(nodeDir, "data", "priv_validator_state.json"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(nodeDir, "config", "node_key.json"))
		assert.NoError(t, err)
	})

	// 测试genesis未设置的情况
	t.Run("genesis未设置的情况", func(t *testing.T) {
		err := peerConfig.SaveFiles()
		assert.Error(t, err)
	})
}

// TestUpdatePort 测试updatePort函数
func TestUpdatePort(t *testing.T) {
	// 测试正常情况
	t.Run("正常情况", func(t *testing.T) {
		urlStr := "tcp://0.0.0.0:26657"
		newPort := uint16(26658)

		result, err := updatePort(urlStr, newPort)
		assert.NoError(t, err)
		assert.Equal(t, "tcp://0.0.0.0:26658", result)
	})

	// 测试端口为0的情况
	t.Run("端口为0的情况", func(t *testing.T) {
		urlStr := "tcp://0.0.0.0:26657"
		newPort := uint16(0)

		result, err := updatePort(urlStr, newPort)
		assert.NoError(t, err)
		assert.Equal(t, "tcp://0.0.0.0:26657", result)
	})

	// 测试端口超出范围的情况
	t.Run("端口超出范围的情况", func(t *testing.T) {
		urlStr := "tcp://0.0.0.0:26657"
		newPort := uint16(65535)

		result, err := updatePort(urlStr, newPort)
		assert.EqualError(t, err, "port 65536 is too large")
		assert.Equal(t, "", result)
	})

	// 测试URL格式错误的情况
	t.Run("URL格式错误的情况", func(t *testing.T) {
		urlStr := "invalid-url"
		newPort := uint16(26658)

		result, err := updatePort(urlStr, newPort)
		assert.Error(t, err)
		assert.Equal(t, "", result)
	})
}
