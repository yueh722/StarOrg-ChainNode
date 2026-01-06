package lib

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	cometbftConfig "github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/crypto"
	"github.com/cometbft/cometbft/v2/libs/log"
	"github.com/cometbft/cometbft/v2/privval"
	"github.com/cometbft/cometbft/v2/types"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var logger = log.NewLogger(os.Stdout)

var nodeDirPerm os.FileMode = 0o755

// homeDir: d:\mychain	[customer path]
//			d:\mychain\peer_config.json	[state file of current node]
// nodeDir: d:\mychain\[nodeID]	[cometbft node path]
// cometbft node config: d:\mychain\[nodeID]\config
// cometbft node config: d:\mychain\[nodeID]\data

type PeerConfig struct {
	config   *cometbftConfig.Config
	genesis  *types.GenesisDoc
	privKey  crypto.PrivKey
	nodeId   string
	homeDir  string
	currPeer *Peer
}

func getConfigFilePath(nodeDir, name string) string {
	return ""
}

func LoadPeerConfig(homeDir, nodeID string) (*PeerConfig, error) {

	nodeDir := filepath.Join(homeDir, nodeID)

	//load cometbft config file
	cmtConfig := cometbftConfig.DefaultConfig()
	viper.SetConfigFile(filepath.Join(nodeDir, "config", "config.toml"))
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(cmtConfig); err != nil {
		return nil, err
	}
	if err := cmtConfig.ValidateBasic(); err != nil {
		return nil, err
	}

	cmtConfig.SetRoot(nodeDir)

	//genesisDoc
	genesisDoc, err := types.GenesisDocFromFile(cmtConfig.GenesisFile())
	if err != nil {
		return nil, err
	}

	//peer-config
	peer := &Peer{}
	peerCfgFilePath := filepath.Join(nodeDir, peerConfigFile)
	if err := peer.Load(peerCfgFilePath); err != nil {
		return nil, err
	}

	return &PeerConfig{
		config:   cmtConfig,
		genesis:  genesisDoc,
		currPeer: peer,
		homeDir:  homeDir,
		nodeId:   peer.NodeId,
		privKey:  peer.PrivateKey(),
	}, nil
}

func NewPeerConfig(currPeer *Peer, persistentPeers, homeDir string) (*PeerConfig, error) {
	var err error

	if currPeer == nil {
		return nil, errors.New("currPeer is nil")
	}

	if homeDir == "" {
		homeDir = "./"
	}
	nodeDir := filepath.Join(homeDir, currPeer.NodeId)

	defer func() {
		if err != nil {
			_ = os.RemoveAll(nodeDir)
		}
	}()

	config := cometbftConfig.DefaultConfig()

	config.SetRoot(nodeDir)

	// Overwrite default config.
	config.RPC.ListenAddress, err = updatePort(config.RPC.ListenAddress, currPeer.RPCPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update rpc listen address")
	}

	config.BaseConfig.ProxyApp, err = updatePort(config.BaseConfig.ProxyApp, currPeer.ABCIPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update ABCI port")
	}

	config.P2P.ListenAddress, err = updatePort(config.P2P.ListenAddress, currPeer.P2PPort)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update P2P listen address")
	}

	config.P2P.AddrBookStrict = false
	config.P2P.AllowDuplicateIP = true
	config.P2P.PersistentPeers = persistentPeers
	config.Moniker = currPeer.NodeName

	// don't create empty block
	config.Consensus.CreateEmptyBlocks = false

	//config.RPC.TLSCertFile = ""
	//config.RPC.TLSKeyFile = ""

	return &PeerConfig{
		config:   config,
		homeDir:  homeDir,
		privKey:  currPeer.PrivateKey(),
		nodeId:   currPeer.NodeId,
		genesis:  nil,
		currPeer: currPeer,
	}, nil
}

func (cfg *PeerConfig) NewGensis(chainId string, genesisTime time.Time, validators []types.GenesisValidator) error {

	// Generate genesis doc from generated validators
	cfg.genesis = &types.GenesisDoc{
		ChainID:         "chain-" + chainId,
		ConsensusParams: types.DefaultConsensusParams(),
		GenesisTime:     genesisTime,
		InitialHeight:   0,
		Validators:      validators,
	}

	return nil
}

func (cfg *PeerConfig) SetGenesis(genesisDoc *types.GenesisDoc, validators []types.GenesisValidator) error {

	if validators != nil && len(validators) > 0 {
		genesisDoc.Validators = validators
	}

	if err := genesisDoc.ValidateAndComplete(); err != nil {
		return errors.New("genesisDoc is corrupt")
	}

	cfg.genesis = genesisDoc
	return nil
}

func (cfg *PeerConfig) UpdateChainConfig(peers PeerList) error {
	if err := cfg.SetGenesis(cfg.genesis, peers.BuildValidators()); err != nil {
		return err
	}

	cfg.config.P2P.PersistentPeers = peers.BuildPersistentPeers()

	nodeDir := cfg.GetNodeDir()
	config := cfg.config

	//save genesis.json
	if err := cfg.genesis.SaveAs(filepath.Join(nodeDir, config.BaseConfig.Genesis)); err != nil {
		return err
	}

	//save config.toml
	cometbftConfig.WriteConfigFile(filepath.Join(nodeDir, "config", "config.toml"), cfg.config)

	return nil
}

func (cfg *PeerConfig) GetNodeDir() string {
	return filepath.Join(cfg.homeDir, cfg.nodeId)
}

func (cfg *PeerConfig) GetPeerConfigFilePath() string {
	return filepath.Join(cfg.homeDir, peerConfigFile)
}

func (cfg *PeerConfig) SaveFiles() error {
	var err error

	nodeDir := cfg.GetNodeDir()

	defer func() {
		if err != nil {
			_ = os.RemoveAll(nodeDir)
		}
	}()

	err = os.MkdirAll(filepath.Join(nodeDir, "config"), nodeDirPerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(nodeDir, "data"), nodeDirPerm)
	if err != nil {
		return err
	}

	// ### Write genesis file.
	config := cometbftConfig.DefaultConfig()
	if err := cfg.genesis.SaveAs(filepath.Join(nodeDir, config.BaseConfig.Genesis)); err != nil {
		return err
	}

	// ### Write config file.
	cometbftConfig.WriteConfigFile(filepath.Join(nodeDir, "config", "config.toml"), cfg.config)

	// ### Write priv_validator_key file and priv_validator_state file.
	if err := cfg.writeValidatorFiles(); err != nil {
		return err
	}

	// ### Write node_key file.
	if err := cfg.writeNodeKeyFile(); err != nil {
		return err
	}

	// ### Write peer config file
	if err := cfg.currPeer.Save(cfg.GetPeerConfigFilePath()); err != nil {
		return err
	}

	return nil
}

func (cfg *PeerConfig) writeNodeKeyFile() error {
	nodeKeyFile := cfg.config.NodeKeyFile()

	nodeKey := NodekeyFromBase64(cfg.currPeer.NodeKey)

	if err := nodeKey.SaveAs(nodeKeyFile); err != nil {
		return err
	}

	logger.Info("Generated node key", "path", nodeKeyFile)
	return nil
}

// ed25519.KeyType
func genPrivKeyFromString(prvkey crypto.PrivKey) func() (crypto.PrivKey, error) {
	return func() (crypto.PrivKey, error) {
		return prvkey, nil
	}
}

func (cfg *PeerConfig) writeValidatorFiles() error {

	privValKeyFile := cfg.config.PrivValidatorKeyFile()
	privValStateFile := cfg.config.PrivValidatorStateFile()

	var pv *privval.FilePV
	var err error
	pv, err = privval.GenFilePV(privValKeyFile, privValStateFile, genPrivKeyFromString(cfg.privKey))
	if err != nil {
		return fmt.Errorf("can't generate file pv: %w", err)
	}
	pv.Save()
	logger.Info("Generated private validator", "keyFile", privValKeyFile,
		"stateFile", privValStateFile)

	return nil
}

func updatePort(urlStr string, newPort uint16) (string, error) {
	if newPort == 0 {
		return urlStr, nil
	}

	if newPort > 65535 {
		return "", fmt.Errorf("port %d is too large", newPort)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	u.Host = u.Hostname() + ":" + strconv.Itoa(int(newPort))

	return u.String(), nil
}

func UpdateHost(urlStr string, newHost string) (string, error) {
	if newHost == "" || newHost == "0.0.0.0" {
		return urlStr, nil
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	u.Host = u.Hostname() + ":" + u.Port()
	return u.String(), nil
}
