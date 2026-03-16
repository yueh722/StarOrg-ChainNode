package setupTypes

import (
	"chain/lib"
	"encoding/json"

	"github.com/cometbft/cometbft/v2/types"
)

type DelegatorInfo struct {
	App        string
	StatusNode StatusNode
	DeviceID   string
}

// Management
// method: Subscription
type SetupInitializeChainNode struct {
	NodeID     string           `json:"node_id,omitempty"`
	NodeArray  lib.PeerList     `json:"node_array,omitempty"`
	GenesisDoc types.GenesisDoc `json:"genesis_doc"`
}

// method: Subscription
type SetupUpdateChainCfg struct {
	NodeArray lib.PeerList `json:"node_array,omitempty"`
}

type SetupAddValidator struct {
	lib.AddValidatorReq `json:"add_validator_req"`
}

type SetupRemoveValidator struct {
	lib.RemoveValidatorReq `json:"remove_validator_req"`
}

type SetupPurgeChain struct{}

type NodeAction string

const (
	NodeStart   NodeAction = "@start"
	NodeStop    NodeAction = "@stop"
	NodeRestart NodeAction = "@restart"
	NodeReset   NodeAction = "@reset"
	NodePurge   NodeAction = "@purge"
)

// method: Subscription
type SetupControlNode struct {
	NodeID string     `json:"node_id,omitempty"`
	Action NodeAction `json:"action,omitempty"`
}

type SetupPing struct {
	NodeID    string `json:"node_id,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type HostInfo struct {
	HostName string   `json:"host_name"`
	IpAddrs  []string `json:"ip_addresses"`
}

// Status report
// each client will report node information to server freqently
// method: RPC
type StatusNode struct {
	lib.NodeInfo `json:"node_info"`
	HostInfo     `json:"host_info"`
	Online       bool   `json:"online"`
	Healthy      bool   `json:"healthy,omitempty"`
	LastCheck    string `json:"last_check,omitempty"`
	StartedTime  string `json:"started_time,omitempty"`
	Initialized  bool   `json:"initialized,omitempty"`
}

type StatusAgent struct {
	Catalog   string `json:"catalog,omitempty"`
	NodeID    string `json:"node_id,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
	Status    string `json:"status,omitempty"`
	IssuedAt  string `json:"issued_at,omitempty"`
}

type StatusCometbftProcess struct {
	Catalog  string `json:"catalog,omitempty"`
	NodeID   string `json:"node_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Error    string `json:"error,omitempty"`
	IssuedAt string `json:"issued_at,omitempty"`
}

// storage report of host running node
// each client will report storage information of host to server freqently
// method: RPC
type StatusStorage struct {
	Path        string `json:"path,omitempty"`
	TotalSpace  string `json:"total_space,omitempty"`
	FreeSpace   string `json:"free_space,omitempty"`
	UsedSpace   string `json:"used_space,omitempty"`
	UsedPercent string `json:"used_percent,omitempty"`
}

type CmdType string

const (
	CmdPing                CmdType = "setup_ping"
	CmdInitializeChainNode CmdType = "setup_initialize_chain_node"
	CmdUpdateChainCfg      CmdType = "setup_update_chain_cfg"
	CmdAddValidator        CmdType = "setup_add_validator"
	CmdRemoveValidator     CmdType = "setup_remove_validator"
	CmdControlNode         CmdType = "setup_control_node"
	CmdPurgeChain          CmdType = "setup_purge_chain"
)

type CommandEnvelope struct {
	CommandID string          `json:"command_id"`
	Cmd       CmdType         `json:"cmd"`     // e.g. "setup_initialize_chain_node", "setup_control_node"
	NodeID    string          `json:"node_id"` // target node for this client
	Payload   json.RawMessage `json:"payload"`
	IssuedAt  string          `json:"issued_at"`
}
