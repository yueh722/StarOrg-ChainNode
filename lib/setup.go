package lib

import (
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/spf13/viper"
)

const peerConfigFile = "peer_config.json"

type NodeInfo struct {
	NodeName       string `json:"node_name,omitempty"`
	NodeID         string `json:"node_id,omitempty"`
	NodeAddress    string `json:"node_address,omitempty"`
	ProxyApp       string `json:"proxy_app,omitempty"`        //proxy_app: default 26658
	RPCAddr        string `json:"rpc_addr,omitempty"`         //rcp: default 26657
	WebServiceAddr string `json:"web_service_addr,omitempty"` //webapi: default 8080
	Validator      bool   `json:"validator,omitempty"`
}

func GenesisTimeFromUnix(nsec int64) time.Time {
	return cmttime.Canonical(time.Unix(0, nsec))
}

func InitializeChainNode(genesisDoc *types.GenesisDoc, peers PeerList, nodeId, nodeDir string) error {

	currPeer := peers.GetPeerByNodeId(nodeId)
	if currPeer == nil {
		return fmt.Errorf("information of peer %s is required", nodeId)
	}

	peerConfig, err := NewPeerConfig(currPeer, peers.BuildPersistentPeers(), nodeDir)
	if err != nil {
		return err
	}

	var validators []types.GenesisValidator
	if currPeer.IsValidator {
		//update validators of genesisDoc
		validators = peers.BuildValidators()
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

func UpdateChainCfg(peers PeerList, homeDir, nodeID string) error {
	//load config
	peerConfig, err := LoadPeerConfig(homeDir, nodeID)
	if err != nil {
		return err
	}

	return peerConfig.UpdateChainConfig(peers)
}

func AddChainNode(genesisDoc *types.GenesisDoc, peers PeerList, nodeId, nodeDir string) error {

	currPeer := peers.GetPeerByNodeId(nodeId)
	if currPeer == nil {
		return fmt.Errorf("information of peer %s is required", nodeId)
	}

	peerConfig, err := NewPeerConfig(currPeer, peers.BuildPersistentPeers(), nodeDir)
	if err != nil {
		return err
	}

	var validators []types.GenesisValidator
	if currPeer.IsValidator {
		//update validators of genesisDoc
		validators = peers.BuildValidators()
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
	runtime     *NodeRuntime
	NodeLogger  *log.Logger
	AppLogger   *log.Logger
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

func StartNode(nodeID string, opts ...CtlOption) NodeRuntime {
	arg := NodeArg{}
	for _, opt := range opts {
		opt(&arg)
	}

	return StartNodeWithArg(nodeID, arg)
}

func StopNode(runtime NodeRuntime) error {
	if runtime == nil {
		return nil
	}

	runtime.KillNode()
	return nil
}

func RestartNode(nodeID string, opts ...CtlOption) NodeRuntime {
	return nil
}

func CheckChainNode(nodeId, nodeDir string) bool {

	return false
}

func ResetChainNode(homeDir, nodeID string) (*NodeInfo, error) {
	nodeDir := filepath.Join(homeDir, nodeID)
	err := os.RemoveAll(nodeDir)
	if err != nil {
		return nil, fmt.Errorf("remove chain node path %v failed: %v", nodeDir, err)
	}

	nodeInfo, err := ResetPeerConfig(homeDir, nodeID)
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}

func GetNodeInfoFromCometbftConfig(nodeDir string) (*NodeInfo, error) {

	// load config from config.toml in homeDir
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
	RPCAddress, err := UpdateHost(cmtcfg.RPC.ListenAddress, nodeHost)
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

func GetNodeInfo(homeDir string) (*NodeInfo, error) {
	peer := &Peer{}

	if err := peer.Load(filepath.Join(homeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	return PeerInfoToNodeInfo(peer), nil
}

func InitializeNewNodeID(homeDir string) (*NodeInfo, error) {
	peer := &Peer{
		NodeId: NewNodeId(),
	}

	if err := peer.Save(filepath.Join(homeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	if err := os.Mkdir(filepath.Join(homeDir, peer.NodeId), os.ModePerm); err != nil {
		return nil, err
	}

	return PeerInfoToNodeInfo(peer), nil
}

func ResetPeerConfig(homeDir, nodeID string) (*NodeInfo, error) {
	peer := &Peer{
		NodeId: nodeID,
	}

	if err := peer.Save(filepath.Join(homeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	if err := os.Mkdir(filepath.Join(homeDir, peer.NodeId), os.ModePerm); err != nil {
		return nil, err
	}

	return PeerInfoToNodeInfo(peer), nil
}

func PeerInfoToNodeInfo(peer *Peer) *NodeInfo {
	return &NodeInfo{
		NodeName:       peer.NodeName,
		NodeID:         peer.NodeId,
		NodeAddress:    peer.NodeAddress,
		ProxyApp:       fmt.Sprintf("tcp://%s:%d", peer.NodeAddress, peer.ABCIPort),
		RPCAddr:        peer.ToRPCAddr(),
		WebServiceAddr: fmt.Sprintf("%s:%d", peer.NodeAddress, peer.WebServicePort),
		Validator:      peer.IsValidator,
	}
}
