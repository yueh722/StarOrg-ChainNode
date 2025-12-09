package internal

import (
	"errors"
	"fmt"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/spf13/viper"
	"path/filepath"
	"time"

	"github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
)

type NodeServerInfo struct {
	NodeName  string
	RPCAddr   string
	GRPCAddr  string
	ProxyApp  string
	Validator bool
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

func GetNodeRPCServer(nodeDir string) (*NodeServerInfo, error) {

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

	//update listen address with node address

	genesisDoc, err := types.GenesisDocFromFile(cmtcfg.GenesisFile())
	if err != nil {
		return nil, err
	}

	_, isValidator := slice.FindBy(genesisDoc.Validators, func(index int, item types.GenesisValidator) bool {
		return item.Name == cmtcfg.Moniker
	})

	return &NodeServerInfo{
		NodeName:  cmtcfg.Moniker,
		RPCAddr:   cmtcfg.RPC.ListenAddress,
		GRPCAddr:  cmtcfg.GRPC.ListenAddress,
		ProxyApp:  cmtcfg.ProxyApp,
		Validator: isValidator,
	}, nil
}
