package cmd

import (
	"NodeCLI/internal"
	"fmt"
	"github.com/spf13/cobra"
)

// when you want to add a new node into a exist chain that already running for a while. you need to use
// command 'add-node' for the new node.

/*

https://docs.cometbft.com/v0.38/core/using-cometbft

### Adding a Non-Validator
Adding a non-validator is simple. Just copy the original genesis.json to ~/.cometbft/config on the new machine and start the node, specifying seeds or persistent peers as necessary.
If no seeds or persistent peers are specified, the node won’t make any blocks, because it’s not a validator, and it won’t hear about any blocks, because it’s not connected to the other peer.

### Adding a Validator
The easiest way to add new validators is to do it in the genesis.json, before starting the network.
For instance, we could make a new priv_validator_key.json, and copy it’s pub_key into the above genesis.

Update the genesis.json in ~/.cometbft/config.
Copy the genesis file and the new priv_validator_key.json to the ~/.cometbft/config on a new machine.

Now run cometbft node on both machines, and use either --p2p.persistent_peers or the /dial_peers to get them to peer up.
*/

type AddNodeArg struct {
	NodeId string
	//json string that includes private key, p2p address of all nodes
	Peers      string
	GenesisDoc string
}

var addNodeArg = &AddNodeArg{}

var AddNodeFileCmd = &cobra.Command{
	Use:   "add-node",
	Short: "Initialize a node and join it into a CometBFT private chain",
	Long: `add-node will create one node.

Example:

	nodecli add-node --home .\node
	`,
	RunE: addnodeFiles,
}

func CreateAddNodeFileCommand() *cobra.Command {
	cmd := AddNodeFileCmd
	arg := addNodeArg

	cmd.Flags().StringVar(&arg.NodeId, "node-id", "", "node id")

	cmd.Flags().StringVar(&arg.Peers, "peers", "",
		"peers that will be used to sync block between nodes.")

	cmd.Flags().StringVar(&arg.GenesisDoc, "genesis-doc", "",
		"the chain need genesis-doc to start and sync blocks between nodes.")

	return cmd
}

func addnodeFiles(*cobra.Command, []string) error {

	// ### Parse peer addresses.
	peerList := ConvertJsonToPeers(addNodeArg.Peers)
	if len(peerList) == 0 {
		return fmt.Errorf("no peer found, please provide all peers information that includes private key, id, name and p2p address.")
	}

	currPeer := peerList.GetPeerByNodeId(addNodeArg.NodeId)

	peerConfig, err := internal.NewPeerConfig(currPeer, peerList.BuildPersistentPeers(), "")
	if err != nil {
		return err
	}

	if err := peerConfig.SetGenesis(addNodeArg.GenesisDoc); err != nil {
		return err
	}

	if err := peerConfig.SaveFiles(); err != nil {
		return err
	}

	fmt.Printf("Successfully initialized node '%v' directories\n", currPeer.NodeName)

	return nil
}
