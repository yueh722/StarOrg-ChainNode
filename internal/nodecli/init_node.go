package nodecli

// when initialize a chain from scratch, we need to use command 'init-node' to initialize node one by one.

import (
	"chain/lib"
	"fmt"

	cfg "github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/privval"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/spf13/cobra"
)

type NodeInitializationArguments struct {
	chainId     string
	genesisTime int64
	nodeId      string

	//JSON string that includes private key, p2p address of all nodes
	peers string
}

const (
	defaultP2PPort        uint16 = 26656
	defaultRPCPort        uint16 = 26657
	defaultABCIPort       uint16 = 26658
	defaultWebServicePort uint16 = 8080
)

var (
	home         string
	stdoutFormat string
)

var RootCmd = &cobra.Command{
	Use:   "NodeCLI",
	Short: "Command line utilize for cometbft node",
	Long:  `Command line utilize for cometbft node`,
}

var nodeInitArgs = &NodeInitializationArguments{}

func CreateInitNodeFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-node",
		Short: "Initialize a node for a CometBFT private chain",
		Long: `init-node will create one node.

Example:

	nodecli init-node [subcommand]
	`,
		RunE: initnodeFiles,
	}

	arg := nodeInitArgs

	cmd.Flags().StringVar(&arg.chainId, "chain-id", "",
		"chain id that will be used to initialize a chain")

	cmd.Flags().Int64Var(&arg.genesisTime, "genesis-time", 0, "genesis time")

	cmd.Flags().StringVar(&arg.nodeId, "node-id", "", "node id")
	cmd.Flags().StringVar(&arg.peers, "peers", "",
		"a JSON string that includes peer information, for example: name, id, private key, server address, rpc port and etc.")

	return cmd
}

func init() {
	RootCmd.Flags().StringVar(&home, "home", "", "directory for current node")
	RootCmd.Flags().StringVar(&stdoutFormat, "stdout-format", "plant", "stdout format. text or JSON")
}

func initnodeFiles(*cobra.Command, []string) error {
	arg := nodeInitArgs

	err := lib.InitializeChainNode(arg.genesisTime, arg.peers, arg.chainId, arg.nodeId, home)
	if err == nil {
		currPeer := lib.ConvertJsonToPeers(arg.peers).GetPeerByNodeId(arg.nodeId)
		var nodeName string
		if currPeer != nil {
			nodeName = currPeer.NodeName
		}
		fmt.Printf("Successfully initialized node '%v' directories\n", nodeName)
	}

	return err
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
			return fmt.Errorf("can't generate file pv: %v", err)
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
