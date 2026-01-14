package common

import (
	"chain/lib"
	"encoding/json"

	"github.com/cometbft/cometbft/v2/types"
)

type DelegatorInfo struct {
	App        string
	StatusNode StatusNode
}

const DelegatorApp = "NodeProxyD_"

// Management
// method: Subscription
type SetupInitializeChainNode struct {
	NodeID     string
	NodeArray  lib.PeerList
	GenesisDoc types.GenesisDoc
}

// method: Subscription
type SetupUpdateChainCfg struct {
	NodeArray lib.PeerList
}

type SetupResetChain struct {
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
	lib.NodeInfo `json:"lib_._node_info"`
	Healthy      bool   `json:"healthy,omitempty"`
	LastCheck    int64  `json:"last_check,omitempty"`
	StartedTime  string `json:"started_time,omitempty"`
	Initialized  bool   `json:"initialized,omitempty"`
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
	CmdUpdateChainCfg      CmdType = "setup_update_chain_cfg"
	CmdControlNode         CmdType = "setup_control_node"
	CmdResetChain          CmdType = "setup_reset_chain"
	CmdPurgeData           CmdType = "setup_purge_data"
)

type CommandEnvelope struct {
	CommandID string          `json:"command_id"`
	Cmd       CmdType         `json:"cmd"`     // e.g. "setup_initialize_chain_node", "setup_control_node"
	NodeID    string          `json:"node_id"` // target node for this client
	Payload   json.RawMessage `json:"payload"`
	IssuedAt  int64           `json:"issued_at"`
}
