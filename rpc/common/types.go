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
	NodeId     string
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
	TotalSpace     int64
	FreeSpace      int64
	UsedSpace      int64
	FreePercentage int64
}

type CmdType string

const (
	CmdInitializeChainNode CmdType = "setup_initialize_chain_node"
	CmdControlNode         CmdType = "setup_control_node"
)

type CommandEnvelope struct {
	CommandID string          `json:"command_id"`
	Cmd       CmdType         `json:"cmd"`     // e.g. "setup_initialize_chain_node", "setup_control_node"
	NodeID    string          `json:"node_id"` // target node for this client
	Payload   json.RawMessage `json:"payload"`
	IssuedAt  int64           `json:"issued_at"`
}

type CommandStatus string

const (
	CmdAccepted CommandStatus = "accepted" // received, will execute
	CmdRunning  CommandStatus = "running"  // executing (optional progress updates)
	CmdSuccess  CommandStatus = "success"
	CmdFailed   CommandStatus = "failed"
)

type CommandResult struct {
	CommandID string        `json:"command_id"`
	Cmd       string        `json:"cmd"`
	NodeID    string        `json:"node_id"`
	Status    CommandStatus `json:"status"`

	Progress float64 `json:"progress,omitempty"` // 0~100 (optional)
	Message  string  `json:"message,omitempty"`  // human readable
	Error    string  `json:"error,omitempty"`    // non-empty when failed

	ReportedAt int64 `json:"reported_at"`           // report time (always)
	StartedAt  int64 `json:"started_at,omitempty"`  // optional
	FinishedAt int64 `json:"finished_at,omitempty"` // optional

	// optional: structured output, logs, extra metadata, etc.
	Output json.RawMessage `json:"output,omitempty"`
}
