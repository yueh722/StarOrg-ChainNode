package cmd

import (
	"NodeCLI/internal"
	"encoding/json"
	"fmt"
	cfg "github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/crypto"
	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/libs/log"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/privval"
	"github.com/cometbft/cometbft/v2/types"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

var (
	chainId string
	nodeId  string
	//nodeAddress    string // ip or domain
	//nodePrvKey     string
	peers          string //json string that includes private key, p2p address of all nodes
	p2pPort        uint16
	rpcPort        uint16
	abciPort       uint16
	webServicePort uint16

	logger = log.NewLogger(os.Stdout)
)

const (
	defaultP2PPort        uint16 = 26656
	defaultRPCPort        uint16 = 26657
	defaultABCIPort       uint16 = 26658
	defaultWebServicePort uint16 = 8080

	nodeDirPerm = 0o755
)

var InitNodeFilesCmd = &cobra.Command{
	Use:   "init-node",
	Short: "Initialize a node for a CometBFT private chain",
	Long: `init-node will create one node.

Example:

	nodecli init-node --home .\node
	`,
	RunE: initnodeFiles,
}

func init() {

	/*
		a.	ChainId			alpha.com
		b.	NodeName		node01
		c.	NodeDomain		node01.alpha.com. need to be resolved by dns
		d.	NodeP2P			node-id(encode of pub address)@IP1:26656:node-id-2@IP2:P2P-PORT
		e.	Prvkey			private key string
	*/

	InitNodeFilesCmd.Flags().StringVar(&chainId, "chain-id", "",
		"chain id that will be used to initialize a chain")

	//InitNodeFilesCmd.Flags().StringVar(&nodeName, "node-name", "", "node name ")

	//InitNodeFilesCmd.Flags().StringVar(&nodeAddress, "node-address", "", "node domain or ip address")

	//InitNodeFilesCmd.Flags().StringVar(&nodePrvKey, "node-prvkey", "", "node private key")

	InitNodeFilesCmd.Flags().StringVar(&peers, "peers", "",
		"peers that will be used to sync block between nodes.")

	InitNodeFilesCmd.Flags().Uint16Var(&p2pPort, "p2p-port", defaultP2PPort,
		"p2p port for Cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&rpcPort, "rpc-port", defaultRPCPort,
		"rpc port for Cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&abciPort, "abci-port", defaultABCIPort,
		"application blockchain interface port for cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&webServicePort, "web-service-port", defaultWebServicePort,
		"web service port that cometbft app will listen.")
}

func initnodeFiles(*cobra.Command, []string) error {

	// ### Parse peer addresses.
	peerList := ConvertJsonToPeers(peers)
	if len(peerList) == 0 {
		return fmt.Errorf("no peer found, please provide all peers information that includes private key, id, name and p2p address.")
	}

	currPeer := peerList.GetPeerByNodeId(nodeId)

	config := cfg.DefaultConfig()

	nodeDir := "./" + currPeer.NodeName

	config.SetRoot(nodeDir)

	err := os.MkdirAll(filepath.Join(nodeDir, "config"), nodeDirPerm)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return err
	}
	err = os.MkdirAll(filepath.Join(nodeDir, "data"), nodeDirPerm)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return err
	}

	// ### Write private key of node
	if err = initFilesWithConfig(config, currPeer.PrivKey, nodeId); err != nil {
		return err
	}

	// Generate genesis doc from generated validators
	genDoc := &types.GenesisDoc{
		ChainID:         "chain-" + chainId,
		ConsensusParams: types.DefaultConsensusParams(),
		GenesisTime:     cmttime.Now(),
		InitialHeight:   0,
		Validators:      peerList.BuildValidators(),
	}

	// ### Write genesis file.
	if err := genDoc.SaveAs(filepath.Join(nodeDir, config.BaseConfig.Genesis)); err != nil {
		_ = os.RemoveAll(nodeDir)
		return err
	}

	// Gather persistent peer addresses.
	//persistentPeers, err := persistentPeersString(config)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return err
	}

	// Overwrite default config.
	config.RPC.ListenAddress, err = updatePort(config.RPC.ListenAddress, rpcPort)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return errors.Wrap(err, "failed to update rpc listen address")
	}

	config.BaseConfig.ProxyApp, err = updatePort(config.BaseConfig.ProxyApp, abciPort)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return errors.Wrap(err, "failed to update ABCI port")
	}

	config.P2P.ListenAddress, err = updatePort(config.P2P.ListenAddress, p2pPort)
	if err != nil {
		_ = os.RemoveAll(nodeDir)
		return errors.Wrap(err, "failed to update P2P listen address")
	}

	config.P2P.AddrBookStrict = false
	config.P2P.AllowDuplicateIP = true
	config.P2P.PersistentPeers = peerList.BuildPersistentPeers()
	config.Moniker = currPeer.NodeName

	// ### Write config file.
	cfg.WriteConfigFile(filepath.Join(nodeDir, "config", "config.toml"), config)

	fmt.Printf("Successfully initialized node '%v' directories\n", currPeer.NodeName)

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

// ed25519.KeyType
func genPrivKeyFromString(prvkey string) func() (crypto.PrivKey, error) {
	return func() (crypto.PrivKey, error) {
		return ed25519.PrivKey(prvkey), nil
	}
}

func initFilesWithConfig(config *cfg.Config, prvkey, nodeId string) error {
	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if fileutil.IsExist(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
		logger.Info("Found private validator", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	} else {
		var err error
		pv, err = privval.GenFilePV(privValKeyFile, privValStateFile, genPrivKeyFromString(prvkey))
		if err != nil {
			return fmt.Errorf("can't generate file pv: %w", err)
		}
		pv.Save()
		logger.Info("Generated private validator", "keyFile", privValKeyFile,
			"stateFile", privValStateFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if fileutil.IsExist(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {

		nodeKey := &p2p.NodeKey{
			PrivKey: ed25519.PrivKey(nodeId),
		}

		if err := nodeKey.SaveAs(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	return nil
}

func ConvertJsonToPeers(jsonStr string) internal.PeerList {
	var peers []internal.Peer
	err := json.Unmarshal([]byte(jsonStr), &peers)
	if err != nil {
		fmt.Println("Error while parsing json string", err)
	}
	return peers
}
