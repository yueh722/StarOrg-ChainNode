package lib

import (
	"chain/lib/queuestore"
	log "chain/lib/zlog"
	"chain/utils"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/spf13/viper"
)

const peerConfigFile = "peer_config.json"

type NodeInfo struct {
	NodeName             string `json:"node_name,omitempty"`
	NodeID               string `json:"node_id,omitempty"`
	NodeAddress          string `json:"node_address,omitempty"`
	ProxyApp             string `json:"proxy_app,omitempty"`
	RPCAddr              string `json:"rpc_addr,omitempty"`
	WebServiceListenAddr string `json:"web_service_listen_addr,omitempty"`
	Validator            bool   `json:"validator,omitempty"`
}

func GenesisTimeFromUnix(nsec int64) time.Time {
	return cmttime.Canonical(time.Unix(0, nsec))
}

func InitializeChainNode(genesisDoc *types.GenesisDoc, peers PeerList, nodeId, homeDir string) error {

	currPeer := peers.GetPeerByNodeId(nodeId)
	if !currPeer.Valid() {
		return fmt.Errorf("information of peer %s is invalid", nodeId)
	}

	peerConfig, err := NewPeerConfig(currPeer, peers.BuildPersistentPeers(), homeDir)
	if err != nil {
		return err
	}

	var validators []types.GenesisValidator
	if currPeer.IsValidator && len(genesisDoc.Validators) == 0 {
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
	if !currPeer.Valid() {
		return fmt.Errorf("information of peer %s is invalid", nodeId)
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
	LogFilePath    string
	LogStore       queuestore.QueueStore
	NodeInfo       *NodeInfo
	BinPath        string
	NodeBaseDir    string
	runtime        *NodeRuntime
	NodeNotifyFunc NotifyFunc
	AppNotifyFunc  NotifyFunc
	NodeLogger     *log.Logger
	AppLogger      *log.Logger
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

	return runtime.StopNode()
}

func KillNode(nodeID string) error {
	//kill all node and app processes
	var errs error

	if err := utils.KillProcessByProcNameAndCmdline(AppFileName, nodeID); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to kill app process, err: %v", err))
	}

	if err := utils.KillProcessByProcNameAndCmdline(NodeFileName, nodeID); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to kill node process, err: ", err))
	}

	return errs
}

func RestartNode(nodeID string, opts ...CtlOption) NodeRuntime {
	return nil
}

func CheckChainNode(nodeId, nodeDir string) bool {

	return false
}

func PurgeChainNode(homeDir, nodeID, nodeName string) (*NodeInfo, error) {
	nodeDir := filepath.Join(homeDir, nodeID)
	err := os.RemoveAll(nodeDir)
	if err != nil {
		return nil, fmt.Errorf("remove chain node path %v failed: %v", nodeDir, err)
	}

	nodeInfo, err := ResetPeerConfig(homeDir, nodeID, nodeName)
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}

func IsNodeOriginalValidator(nodeDir string) (bool, error) {

	// load config from config.toml in nodeDir
	viper.AddConfigPath(filepath.Join(nodeDir, "config"))
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		return false, err
	}

	cmtcfg := config.DefaultConfig()
	if err := viper.Unmarshal(cmtcfg); err != nil {
		return false, err
	}

	cmtcfg.SetRoot(nodeDir)

	if err := cmtcfg.ValidateBasic(); err != nil {
		return false, fmt.Errorf("error in config file: %w", err)
	}

	genesisDoc, err := types.GenesisDocFromFile(cmtcfg.GenesisFile())
	if err != nil {
		return false, err
	}

	_, isValidator := slice.FindBy(genesisDoc.Validators, func(index int, item types.GenesisValidator) bool {
		return item.Name == cmtcfg.Moniker
	})

	return isValidator, nil
}

func GetPeerInfo(homeDir string) (*Peer, error) {
	peer := &Peer{}

	if err := peer.Load(filepath.Join(homeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	return peer, nil
}

func GetNodeInfo(homeDir string) (*NodeInfo, error) {
	peer := &Peer{}

	if err := peer.Load(filepath.Join(homeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	return PeerInfoToNodeInfo(peer), nil
}

func ExtractNodeNameFromAppName(appName string) string {
	if len(appName) <= len(DelegatorApp) {
		return ""
	}
	return appName[len(DelegatorApp):]
}

func InitializeNewNodeID(homeDir, appName string) (*NodeInfo, error) {
	return ResetPeerConfig(homeDir, NewNodeId(), appName)
}

func ResetPeerConfig(homeDir, nodeID, nodeName string) (*NodeInfo, error) {
	if nodeName == "" {
		nodeName = RandomNodeName()
	}

	peer := &Peer{
		NodeId:   nodeID,
		NodeName: nodeName,
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
		NodeName:             peer.NodeName,
		NodeID:               peer.NodeId,
		NodeAddress:          peer.NodeAddress,
		ProxyApp:             peer.ProxyAppAddr(),
		RPCAddr:              peer.RPCAddr(),
		WebServiceListenAddr: peer.WebServiceListenAddr(),
		Validator:            peer.IsValidator,
	}
}

func IsNodeInitialized(homeDir string, nodeInfo *NodeInfo) bool {
	if nodeInfo == nil {
		return false
	}

	if nodeInfo.NodeName == "" ||
		nodeInfo.ProxyApp == "" ||
		nodeInfo.WebServiceListenAddr == "" ||
		nodeInfo.RPCAddr == "" {
		return false
	}

	nodeDir := filepath.Join(homeDir, nodeInfo.NodeID)
	// load config from config.toml in nodeDir
	viper.AddConfigPath(filepath.Join(nodeDir, "config"))
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		//fmt.Println("read cometbft config failure:", err)
		return false
	}

	cmtcfg := config.DefaultConfig()
	if err := viper.Unmarshal(cmtcfg); err != nil {
		//fmt.Println("unmarshal to cometbft config failure:", err)
		return false
	}

	cmtcfg.SetRoot(nodeDir)

	if err := cmtcfg.ValidateBasic(); err != nil {
		//fmt.Println("error in config file:", err)
		return false
	}

	genesisDoc, err := types.GenesisDocFromFile(cmtcfg.GenesisFile())
	if err != nil {
		//fmt.Println("failed to read genesisDoc from file:", err)
		return false
	}

	if err = genesisDoc.ValidateAndComplete(); err != nil {
		//fmt.Println("genesisDoc incorrect:", err)
		return false
	}

	return true
}

func ResetNodeInfo(nodeInfo *NodeInfo) {
	if nodeInfo == nil {
		return
	}

	nodeInfo.NodeAddress = ""
	nodeInfo.WebServiceListenAddr = ""
	nodeInfo.RPCAddr = ""
	nodeInfo.ProxyApp = ""
	nodeInfo.Validator = false
}

func RPCWebsocketAddrs(nodeInfo *NodeInfo) []string {

	var rpcAddr string

	prefixs := []string{
		"http://",
		"https://",
	}
	for _, prefix := range prefixs {
		if strings.HasPrefix(nodeInfo.RPCAddr, prefix) {
			rpcAddr = strings.TrimPrefix(nodeInfo.RPCAddr, prefix)
			break
		}
	}

	if rpcAddr == "" {
		rpcAddr = nodeInfo.RPCAddr
	}

	return []string{
		fmt.Sprintf("ws://%s/websocket", rpcAddr),
		fmt.Sprintf("wss://%s/websocket", rpcAddr),
	}
}

// ResetState removes address book files plus all databases and app data files.
func ResetState(nodeDir string) error {
	dbDir := filepath.Join(nodeDir, "data")
	appDir := filepath.Join(nodeDir, "app")

	blockdb := filepath.Join(dbDir, "blockstore.db")
	state := filepath.Join(dbDir, "state.db")
	wal := filepath.Join(dbDir, "cs.wal")
	evidence := filepath.Join(dbDir, "evidence.db")
	txIndex := filepath.Join(dbDir, "tx_index.db")

	dbFiles := []string{blockdb, state, wal, evidence, txIndex, appDir}
	for _, dbFile := range dbFiles {
		if fileutil.IsExist(dbFile) {
			if err := os.RemoveAll(dbFile); err == nil {
				log.Infof("Removed all %v dir", filepath.Base(dbFile))
			} else {
				return fmt.Errorf("error removing all %v dir, err: %v", filepath.Base(dbFile), err)
			}
		}
	}

	utils.TryCreateDir(dbDir)

	return nil
}
