package nodecli

import (
	"chain/lib"
	"fmt"

	"github.com/spf13/cobra"
)

// when you want to add a new node into an exist chain that already running for a while. you need to use
// command 'add-node' for the new node.

/*

https://docs.cometbft.com/v0.38/core/using-cometbft

### Adding a Non-Validator
Adding a non-validator is simple. Just copy the original genesis.json to ~/.cometbft/config on the new machine and start the node, specifying seeds or persistent peers as necessary.
If no seeds or persistent peers are specified, the node won’t make any blocks, because it’s not a validator, and it won’t hear about any blocks, because it’s not connected to the other peer.

### Adding a Validator
The easiest way to add new validators is to do it in the genesis.json, before starting the network.
For instance, we could make a new priv_validator_key.json, and copy its pub_key into the above genesis.

Update the genesis.json in ~/.cometbft/config.
Copy the genesis file and the new priv_validator_key.json to the ~/.cometbft/config on a new machine.

Now run cometbft node on both machines, and use either --p2p.persistent_peers or the /dial_peers to get them to peer up.
*/

type NodeAddArguments struct {
	NodeId string
	//json string that includes private key, p2p address of all nodes
	Peers      string
	GenesisDoc string
}

var nodeAddArgs = &NodeAddArguments{}

func CreateAddNodeFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-node",
		Short: "Initialize a node and join it into a CometBFT private chain",
		Long: `add-node will create one node.

Example:

	nodecli add-node [subcommand]
	`,
		RunE: addnodeFiles,
	}
	arg := nodeAddArgs

	cmd.Flags().StringVar(&arg.NodeId, "node-id", "", "node id")

	cmd.Flags().StringVar(&arg.Peers, "peers", "",
		"a JSON string that includes peer information, for example: name, id, private key, server address, rpc port and etc..")

	cmd.Flags().StringVar(&arg.GenesisDoc, "genesis-doc", "",
		"a JSON string that includes context of genesis.toml which already exist in config directory of other nodes.")

	return cmd
}

func addnodeFiles(*cobra.Command, []string) error {

	arg := nodeAddArgs
	err := lib.AddChainNode(arg.Peers, arg.GenesisDoc, arg.NodeId, home)
	if err == nil {
		currPeer := lib.ConvertJsonToPeers(arg.Peers).GetPeerByNodeId(arg.NodeId)
		var nodeName string
		if currPeer != nil {
			nodeName = currPeer.NodeName
		}
		fmt.Printf("Successfully initialized and added node '%v' directories\n", nodeName)
	}

	return err
}
