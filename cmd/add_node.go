package cmd

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

var (
	_nodeId string

	//json string that includes private key, p2p address of all nodes
	_peers string

	_genesis string
)
