package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
)

const peerConfigFile = "peer_config.json"

type NodeServerInfo struct {
	NodeName       string
	RPCAddr        string
	WebServiceAddr string
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

func StartNode(nodeID string) error {
	return nil

}

func StopNode(nodeID string) error {
	return nil

}

func RestartNode(nodeID string) error {
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
func GetNodeServer(nodeDir string) (*NodeServerInfo, error) {

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

	return &NodeServerInfo{
		NodeName:  cmtcfg.Moniker,
		RPCAddr:   RPCAddress,
		ProxyApp:  cmtcfg.ProxyApp,
		Validator: isValidator,
	}, nil
}
*/

func GetNodeServer(nodeDir string) (*NodeServerInfo, error) {
	peer := Peer{}

	if err := peer.Load(filepath.Join(nodeDir, peerConfigFile)); err != nil {
		return nil, err
	}

	return &NodeServerInfo{
		NodeName:       peer.NodeName,
		RPCAddr:        fmt.Sprintf("http://%s:%d", peer.NodeAddress, peer.RPCPort),
		WebServiceAddr: fmt.Sprintf("http://%s:%d", peer.NodeAddress, peer.WebServicePort),
		Validator:      peer.IsValidator,
	}, nil
}

//run node with monitor

//run app with monitor

//stop node

//stop app
