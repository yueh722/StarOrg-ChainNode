package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cometbft/cometbft/v2/crypto"
	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/p2p"
	"github.com/cometbft/cometbft/v2/types"
)

type Peer struct {
	NodeName       string `json:"node_name"`
	NodeId         string `json:"node_id"`
	PrivKey        string `json:"priv_key"`
	NodeAddress    string `json:"node_address"`
	P2PPort        uint16 `json:"p2p_port"`
	RPCPort        uint16 `json:"rpc_port"`
	ABCIPort       uint16 `json:"abci_port"`
	WebServicePort uint16 `json:"webservice_port"`
	IsValidator    bool   `json:"is_validator"`
}

func (p *Peer) PrivateKey() crypto.PrivKey {

	if key, err := base64.StdEncoding.DecodeString(p.PrivKey); err == nil {
		return ed25519.PrivKey(key)
	}
	return nil
}

type PeerList []Peer

func (peers PeerList) BuildPersistentPeers() string {
	var persistentPeers []string

	for _, peer := range peers {
		//nodeid@nodeaddress:p2p_port,
		p2pAddr := fmt.Sprintf("%s@%s:%d", peer.NodeId, peer.NodeAddress, peer.P2PPort)
		persistentPeers = append(persistentPeers, p2pAddr)
	}

	return strings.Join(persistentPeers, ",")
}

func (peers PeerList) GetPeerByNodeId(nodeId string) *Peer {
	for _, peer := range peers {
		if peer.NodeId == nodeId {
			return &peer
		}
	}

	return nil
}

func (peers PeerList) GetPeerByNodeName(nodeName string) *Peer {
	for _, peer := range peers {
		if peer.NodeName == nodeName {
			return &peer
		}
	}

	return nil
}

func (peers PeerList) BuildValidators() []types.GenesisValidator {

	genVals := make([]types.GenesisValidator, len(peers))

	for i, peer := range peers {

		if !peer.IsValidator {
			continue
		}

		pv := ed25519.PrivKey(peer.PrivKey)

		pubKey := pv.PubKey()

		genVals[i] = types.GenesisValidator{
			Address: pubKey.Address(),
			PubKey:  pubKey,
			Power:   1,
			Name:    peer.NodeName,
		}
	}

	return genVals
}

func NewNodeId() string {
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	return nodeKey.ID()
}

func NewPrivKey() string {
	prvKey := ed25519.GenPrivKey()

	return base64.StdEncoding.EncodeToString(prvKey.Bytes())
}

func ConvertJsonToPeers(jsonStr string) PeerList {
	var peers []Peer
	err := json.Unmarshal([]byte(jsonStr), &peers)
	if err != nil {
		fmt.Println("Error while parsing json string", err)
	}
	return peers
}
