package lib

import "github.com/duke-git/lancet/v2/slice"

type ArgNodeInfo struct {
	NodeName       string `json:"node_name"`
	NodeId         string `json:"node_id"`
	NodeDomainOrIP string `json:"node_domain_or_ip"`
	P2PPort        uint16 `json:"p2p_port"`
	RPCPort        uint16 `json:"rpc_port"`
	ABCIPort       uint16 `json:"abci_port"`
	WebServicePort uint16 `json:"webservice_port"`
	IsValidator    bool   `json:"is_validator"`
}

func (arg *ArgNodeInfo) ToPeer() Peer {
	return Peer{
		NodeName:       arg.NodeName,
		NodeId:         arg.NodeId,
		PrivKey:        NewPrivKey(),
		NodeKey:        NewNodeKey(),
		NodeAddress:    arg.NodeDomainOrIP,
		P2PPort:        arg.P2PPort,
		RPCPort:        arg.RPCPort,
		ABCIPort:       arg.ABCIPort,
		WebServicePort: arg.WebServicePort,
		IsValidator:    arg.IsValidator,
	}
}

func ArgNodeArrayToPeers(nodes []ArgNodeInfo) PeerList {
	return slice.Map(nodes, func(_ int, node ArgNodeInfo) Peer {
		return node.ToPeer()
	})
}
