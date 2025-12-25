package common

import "encoding/json"

// Management
// method: Subscription
type SetupInitializeChainNode struct {
	ChainID     string
	GenesisTime int64
	NodeID      string
	NodeArray   string
}

// method: Subscription
type SetupAddChainNode struct {
	NodeID     string
	NodeArray  string
	GenesisDoc string
}

type NodeAction string

const (
	NodeStart   NodeAction = "!start"
	NodeStop    NodeAction = "!stop"
	NodeRestart NodeAction = "!restart"
	NodeReset   NodeAction = "!reset"
	NodePurge   NodeAction = "!purge"
)

// method: Subscription
type SetupControlNode struct {
	NodeID string
	Action NodeAction
}

// Status report
// each client will report node information to server freqently
// method: RPC
type StatusNode struct {
	NodeId      string
	NodeName    string
	NodeAddress string
	Validator   bool
	ABCIAddress string
	RPCAddress  string
	Healthy     bool
	LastCheck   int64
}

// storage report of host running node
// each client will report storage information of host to server freqently
// method: RPC
type StatusStorage struct {
	Path        string
	TotalSpace  uint64
	FreeSpace   uint64
	UsedSpace   uint64
	UsedPercent float64
}

type CmdType string

const (
	CmdInitializeChainNode CmdType = "setup_initialize_chain_node"
	CmdAddChainNode        CmdType = "setup_add_chain_node"
	CmdControlNode         CmdType = "setup_control_node"
)

type CommandEnvelope struct {
	CommandID string          `json:"command_id"`
	Cmd       CmdType         `json:"cmd"`     // e.g. "setup_initialize_chain_node", "setup_control_node"
	NodeID    string          `json:"node_id"` // target node for this client
	Payload   json.RawMessage `json:"payload"`
	IssuedAt  int64           `json:"issued_at"`
}
