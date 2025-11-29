package cmd

import (
	"github.com/spf13/cobra"
)

var (
	chainId        string
	NodeName       string
	NodeAddress    string // ip or domain
	NodePrvKey     string
	P2PPeers       string
	P2PPort        uint16
	RPCPort        uint16
	ABCIPort       uint16
	WebServicePort uint16
)

const (
	defaultP2PPort        uint16 = 26656
	defaultRPCPort        uint16 = 26657
	defaultABCIPort       uint16 = 26658
	defaultWebServicePort uint16 = 8080
)

var InitNodeFilesCmd = &cobra.Command{
	Use:   "init-node",
	Short: "Initialize a node for a CometBFT private chain",
	Long: `init-node will create one node.

Example:

	nodecli init-node --home .\node --config .\template
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

	InitNodeFilesCmd.Flags().StringVar(&chainId, "chainId", "",
		"chain id that will be used to initialize a chain")

	InitNodeFilesCmd.Flags().StringVar(&NodeName, "node-name", "",
		"node name ")

	InitNodeFilesCmd.Flags().StringVar(&NodeAddress, "node-address", "",
		"node domain or ip address")

	InitNodeFilesCmd.Flags().StringVar(&NodePrvKey, "node-prvkey", "",
		"node private key")

	InitNodeFilesCmd.Flags().StringVar(&P2PPeers, "p2p-peers", "",
		"p2p peers that will be used to sync block between nodes.")

	InitNodeFilesCmd.Flags().Uint16Var(&P2PPort, "p2p-port", defaultP2PPort,
		"p2p port for Cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&RPCPort, "rpc-port", defaultRPCPort,
		"rpc port for Cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&ABCIPort, "abci-port", defaultABCIPort,
		"application blockchain interface port for cometbft node")

	InitNodeFilesCmd.Flags().Uint16Var(&WebServicePort, "web-service-port", defaultWebServicePort,
		"web service port that cometbft app will listen.")
}

func initnodeFiles(*cobra.Command, []string) error {

	return nil
}
