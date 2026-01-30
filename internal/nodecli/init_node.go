package nodecli

// when initialize a chain from scratch, we need to use command 'init-node' to initialize node one by one.

import (
	"chain/lib"
	"fmt"

	"github.com/cometbft/cometbft/v2/types"
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

	genesisDoc := &types.GenesisDoc{
		GenesisTime: lib.GenesisTimeFromUnix(arg.genesisTime),
		ChainID:     arg.chainId,
	}

	// ### Parse peer addresses.
	peerList := lib.ConvertJsonToPeers(arg.peers)
	if len(peerList) == 0 {
		return fmt.Errorf("peers information is required, please provide all peers information that includes private key, id, name and p2p address")
	}

	err := lib.InitializeChainNode(genesisDoc, peerList, arg.nodeId, home)
	if err == nil {
		currPeer := peerList.GetPeerByNodeId(arg.nodeId)
		var nodeName string
		if currPeer != nil {
			nodeName = currPeer.NodeName
		}
		fmt.Printf("Successfully initialized node '%v' directories\n", nodeName)
	}

	return err
}
