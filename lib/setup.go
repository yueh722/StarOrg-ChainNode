package lib

import (
	"chain/lib/queuestore"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
)

const peerConfigFile = "peer_config.json"

type NodeInfo struct {
	NodeName       string
	NodeID         string
	ProxyApp       string //proxy_app: default 26658
	RPCAddr        string //rcp: default 26657
	WebServiceAddr string //webapi: default 8080
	Validator      bool
}

func GetNodeBaseDir(node string) string {
	return "./"
}

func genesisTimeFromUnix(nsec int64) time.Time {
	return cmttime.Canonical(time.Unix(0, nsec))
}

func InitializeChainNode(genesisTime int64, peers, chainId, nodeId, nodeDir string) error {

	if genesisTime == 0 {
		return errors.New("genesis time is required")
	}

	// check if nodeDir is correct.

	// ### Parse peer addresses.
	peerList := ConvertJsonToPeers(peers)
	if len(peerList) == 0 {
		return fmt.Errorf("peers information is required, please provide all peers information that includes private key, id, name and p2p address")
	}

	currPeer := peerList.GetPeerByNodeId(nodeId)
	if currPeer == nil {
		return fmt.Errorf("information of peer %s is required", nodeId)
	}

	peerConfig, err := NewPeerConfig(currPeer, peerList.BuildPersistentPeers(), nodeDir)
	if err != nil {
		return err
	}

	if err := peerConfig.NewGensis(chainId, genesisTimeFromUnix(genesisTime), peerList.BuildValidators()); err != nil {
		return err
	}

	if err := peerConfig.SaveFiles(); err != nil {
		return err
	}

	return nil
}

func AddChainNode(peers, genesisDoc, nodeId, nodeDir string) error {

	// ### Parse peer addresses.
	peerList := ConvertJsonToPeers(peers)
	if len(peerList) == 0 {
		return fmt.Errorf("no peer found. please provide all peers information that includes private key, id, name and p2p address")
	}

	currPeer := peerList.GetPeerByNodeId(nodeId)
	if currPeer == nil {
		return fmt.Errorf("information of peer %s is required", nodeId)
	}

	peerConfig, err := NewPeerConfig(currPeer, peerList.BuildPersistentPeers(), nodeDir)
	if err != nil {
		return err
	}

	var validators []types.GenesisValidator
	if currPeer.IsValidator {
		//update validators of genesisDoc
		validators = peerList.BuildValidators()
	} else {
		validators = nil
	}

	if err := peerConfig.SetGenesis(genesisDoc, validators); err != nil {
		return err
	}

	if err := peerConfig.SaveFiles(); err != nil {
		return err
	}

	return nil
}

type CtlOption func(*NodeArg)

type NodeArg struct {
	LogFilePath string
	LogStore    queuestore.QueueStore
	NodeInfo    *NodeInfo
	BinPath     string
	NodeBaseDir string
}

func WithLogFile(logFilePath string) CtlOption {
	return func(arg *NodeArg) {
		arg.LogFilePath = logFilePath
	}
}

func WithLogStore(store queuestore.QueueStore) CtlOption {
	return func(arg *NodeArg) {
		arg.LogStore = store
	}
}

func WithNodeArgs(set NodeArg) CtlOption {
	return func(arg *NodeArg) {
		*arg = set
	}
}

func StartNode(nodeID string, opts ...CtlOption) error {
	arg := NodeArg{}
	for _, opt := range opts {
		opt(&arg)
	}

	return StartApp(nodeID, arg)
}

func StopNode(nodeID string, opts ...CtlOption) error {
	return nil
}

func RestartNode(nodeID string, opts ...CtlOption) error {
	return nil
}

func CheckChainNode(nodeId, nodeDir string) bool {

	return false
}

func PurgeChainNode(nodeId, nodeDir string) error {
	err := os.RemoveAll(nodeDir)
	if err != nil {
		return fmt.Errorf("purge chain node path [%v] failed: %v", nodeDir, err)
	}

	return nil
}

/*
func GetNodeInfo(nodeDir string) (*NodeInfo, error) {

	// load config from config.toml in nodeDir
	viper.AddConfigPath(filepath.Join(nodeDir, "config"))
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cmtcfg := config.DefaultConfig()
	if err := viper.Unmarshal(cmtcfg); err != nil {
		return nil, err
	}

	cmtcfg.SetRoot(nodeDir)

	if err := cmtcfg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("error in config file: %w", err)
	}

	genesisDoc, err := types.GenesisDocFromFile(cmtcfg.GenesisFile())
	if err != nil {
		return nil, err
	}

	_, isValidator := slice.FindBy(genesisDoc.Validators, func(index int, item types.GenesisValidator) bool {
		return item.Name == cmtcfg.Moniker
	})

	//update listen address with node address
	nodeKey, err := p2p.LoadNodeKey(cmtcfg.NodeKeyFile())
	if err != nil {
		return nil, err
	}
	nodeHost := NewPersistentPeers(cmtcfg.P2P.PersistentPeers).GetNodeHostById(nodeKey.ID())
	RPCAddress, err := updateHost(cmtcfg.RPC.ListenAddress, nodeHost)
	if err != nil {
		return nil, err
	}

	return &NodeInfo{
		NodeName:  cmtcfg.Moniker,
		RPCAddr:   RPCAddress,
		ProxyApp:  cmtcfg.ProxyApp,
		Validator: isValidator,
	}, nil
}
*/

func GetNodeInfo(nodeDir string) (*NodeInfo, error) {
	peer := &Peer{}

	if err := peer.Load(filepath.Join(nodeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	return &NodeInfo{
		NodeName:       peer.NodeName,
		NodeID:         peer.NodeId,
		ProxyApp:       fmt.Sprintf("tcp://%s:%d", peer.NodeAddress, peer.ABCIPort),
		RPCAddr:        fmt.Sprintf("http://%s:%d", peer.NodeAddress, peer.RPCPort),
		WebServiceAddr: fmt.Sprintf("%s:%d", peer.NodeAddress, peer.WebServicePort),
		Validator:      peer.IsValidator,
	}, nil
}
