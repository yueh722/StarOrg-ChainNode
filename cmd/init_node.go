package cmd

// when initialize a chain from scratch, we need to use command 'init-node' to initialize node one by one.

import (
	"NodeCLI/internal"
	"encoding/json"
	"fmt"
	cfg "github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/privval"
	cmttime "github.com/cometbft/cometbft/v2/types/time"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"time"
)

var (
	chainId     string
	genesisTime int64
	nodeId      string

	//json string that includes private key, p2p address of all nodes
	peers string
)

const (
	defaultP2PPort        uint16 = 26656
	defaultRPCPort        uint16 = 26657
	defaultABCIPort       uint16 = 26658
	defaultWebServicePort uint16 = 8080
)

var RootCmd = &cobra.Command{}

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

	InitNodeFilesCmd.Flags().Int64Var(&genesisTime, "genesis-time", 0, "genesis time")

	InitNodeFilesCmd.Flags().StringVar(&nodeId, "node-id", "", "node id")

	//InitNodeFilesCmd.Flags().StringVar(&nodeName, "node-name", "", "node name ")

	//InitNodeFilesCmd.Flags().StringVar(&nodeAddress, "node-address", "", "node domain or ip address")

	//InitNodeFilesCmd.Flags().StringVar(&nodePrvKey, "node-prvkey", "", "node private key")

	InitNodeFilesCmd.Flags().StringVar(&peers, "peers", "",
		"peers that will be used to sync block between nodes.")

	/*
		InitNodeFilesCmd.Flags().Uint16Var(&p2pPort, "p2p-port", defaultP2PPort,
			"p2p port for Cometbft node")

		InitNodeFilesCmd.Flags().Uint16Var(&rpcPort, "rpc-port", defaultRPCPort,
			"rpc port for Cometbft node")

		InitNodeFilesCmd.Flags().Uint16Var(&abciPort, "abci-port", defaultABCIPort,
			"application blockchain interface port for cometbft node")

		InitNodeFilesCmd.Flags().Uint16Var(&webServicePort, "web-service-port", defaultWebServicePort,
			"web service port that cometbft app will listen.")
	*/

}

func genesisTimeFromUnix(nsec int64) time.Time {
	return cmttime.Canonical(time.Unix(0, nsec))
}

func initnodeFiles(*cobra.Command, []string) error {

	if genesisTime == 0 {
		return errors.New("genesis time is required")
	}

	// ### Parse peer addresses.
	peerList := ConvertJsonToPeers(peers)
	if len(peerList) == 0 {
		return fmt.Errorf("no peer found, please provide all peers information that includes private key, id, name and p2p address.")
	}

	currPeer := peerList.GetPeerByNodeId(nodeId)

	peerConfig, err := internal.NewPeerConfig(currPeer, peerList.BuildPersistentPeers(), "")
	if err != nil {
		return err
	}

	if err := peerConfig.NewGensis(chainId, genesisTimeFromUnix(genesisTime), peerList.BuildValidators()); err != nil {
		return err
	}

	if err := peerConfig.SaveFiles(); err != nil {
		return err
	}

	fmt.Printf("Successfully initialized node '%v' directories\n", currPeer.NodeName)

	return nil
}

func initFilesWithConfig(config *cfg.Config, prvkey, nodeId string) error {
	// private validator
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	var pv *privval.FilePV
	if fileutil.IsExist(privValKeyFile) {
		pv = privval.LoadFilePV(privValKeyFile, privValStateFile)
		//logger.Info("Found private validator", "keyFile", privValKeyFile, "stateFile", privValStateFile)
	} else {
		var err error
		//pv, err = privval.GenFilePV(privValKeyFile, privValStateFile, genPrivKeyFromString(prvkey))
		if err != nil {
			return fmt.Errorf("can't generate file pv: %w", err)
		}
		pv.Save()
		//logger.Info("Generated private validator", "keyFile", privValKeyFile, "stateFile", privValStateFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if fileutil.IsExist(nodeKeyFile) {
		//logger.Info("Found node key", "path", nodeKeyFile)
	} else {

		nodeKey := &p2p.NodeKey{
			PrivKey: ed25519.PrivKey(nodeId),
		}

		if err := nodeKey.SaveAs(nodeKeyFile); err != nil {
			return err
		}
		//logger.Info("Generated node key", "path", nodeKeyFile)
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
