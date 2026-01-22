package lib

import (
	log "chain/lib/zlog"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	NodeKey        string `json:"node_key"` //node_key.json and persistentpeer will use this value
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

func (p *Peer) BuildValidator() types.GenesisValidator {
	pubKey := p.PrivateKey().PubKey()

	return types.GenesisValidator{
		Address: pubKey.Address(),
		PubKey:  pubKey,
		Power:   10,
		Name:    p.NodeName,
	}
}

func (p *Peer) Save(peerFile string) error {

	jsonBytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshal peerConfig at %s: %w", peerFile, err)
	}

	if err := os.MkdirAll(filepath.Dir(peerFile), 0755); err != nil {
		return fmt.Errorf("error making directory of file %s: %w", peerFile, err)
	}

	err = os.WriteFile(peerFile, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing peer config at %s: %w", peerFile, err)
	}

	log.Infof("peer config at %s saved with %v : %v", peerFile, p.NodeId, p.NodeName)

	return nil
}

func (p *Peer) Load(peerFile string) error {
	jsonBlob, err := os.ReadFile(peerFile)
	if err != nil {
		return fmt.Errorf("couldn't read peer config file: %w", err)
	}

	err = json.Unmarshal(jsonBlob, p)
	if err != nil {
		return fmt.Errorf("error unmarshal peer Config at %s: %w", peerFile, err)
	}
	return nil
}

func (p *Peer) PersistentPeerID() string {
	return NodekeyFromBase64(p.NodeKey).ID()
}

func (p *Peer) NodeFileID() string {
	return NodekeyFromNodeID(p.NodeId).ID()
}

func (p *Peer) ToRPCAddr() string {
	// please note that cometbft node will listen rpc port which is always 127.0.0.1
	// ref: config.toml -> [rpc].laddr
	return fmt.Sprintf("http://%s:%d", "127.0.0.1", p.RPCPort)
}

func (p *Peer) ToProxyAppAddr() string {
	return fmt.Sprintf("tcp://%s:%d", p.NodeAddress, p.ABCIPort)
}

func (p *Peer) ToWebServiceAddr() string {
	return fmt.Sprintf("http://%s:%d", p.NodeAddress, p.WebServicePort)
}

func (p *Peer) ToLocalWebServiceAddr() string {
	return fmt.Sprintf("http://localhost:%d", p.WebServicePort)
}

func (p *Peer) ToWebsocketAddrs() []string {
	return []string{
		fmt.Sprintf("ws://%s:%d/websocket", p.NodeAddress, p.RPCPort),
		fmt.Sprintf("wss://%s:%d/websocket", p.NodeAddress, p.RPCPort),
	}
}

func (p *Peer) Valid() bool {
	if p.NodeId == "" ||
		p.NodeKey == "" ||
		p.NodeName == "" ||
		p.PrivKey == "" ||
		p.NodeAddress == "" ||
		p.P2PPort == 0 ||
		p.RPCPort == 0 ||
		p.ABCIPort == 0 ||
		p.WebServicePort == 0 {
		return false
	}

	return true
}

func (p *Peer) ToJson() string {
	if jsonBytes, err := json.Marshal(p); err == nil {
		return string(jsonBytes)
	}

	return ""
}

type PeerList []Peer

func (peers PeerList) BuildPersistentPeers() string {
	var persistentPeers []string

	for _, peer := range peers {
		//peerid@nodeaddress:p2p_port,
		p2pAddr := fmt.Sprintf("%s@%s:%d", peer.PersistentPeerID(), peer.NodeAddress, peer.P2PPort)
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

	return &Peer{}
}

func (peers PeerList) GetPeerByNodeName(nodeName string) *Peer {
	for _, peer := range peers {
		if peer.NodeName == nodeName {
			return &peer
		}
	}

	return &Peer{}
}

func (peers PeerList) BuildValidators() []types.GenesisValidator {

	genVals := make([]types.GenesisValidator, len(peers))

	for i, peer := range peers {
		if !peer.IsValidator {
			continue
		}

		genVals[i] = peer.BuildValidator()
	}

	return genVals
}

func (peers PeerList) ToJson() string {
	if jsonBytes, err := json.Marshal(peers); err == nil {
		return string(jsonBytes)
	}

	return ""
}

func RandomNodeName() string {
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	return nodeKey.ID()[:5]
}

func NewNodeId() string {
	nodeKey := &p2p.NodeKey{
		PrivKey: ed25519.GenPrivKey(),
	}

	//return hex.EncodeToString(nodeKey.PrivKey.Bytes())
	return nodeKey.ID()
}

func NodekeyFromBase64(b64Str string) *p2p.NodeKey {
	//nodeID is a base64 encoded string of private key,
	//convert this private key to nodeKey.

	prvKey, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		log.Fatalf("error decoding node key: %v", err)
	}

	return &p2p.NodeKey{
		PrivKey: ed25519.PrivKey(prvKey),
	}
}

func NodekeyFromNodeID(nodeID string) *p2p.NodeKey {
	//nodeID is a hex encoded string of private key,
	//convert this private key to nodeKey.

	prvKey, err := hex.DecodeString(nodeID)
	if err != nil {
		log.Fatalf("error decoding node key: %v", err)
	}

	return &p2p.NodeKey{
		PrivKey: ed25519.PrivKey(prvKey),
	}
}

func PersistentPeerIDFromNodeId(nodeID string) string {
	//use nodeKey.ID() get Persistent peer ID
	return NodekeyFromNodeID(nodeID).ID()
}

func NewPrivKey() string {
	prvKey := ed25519.GenPrivKey()

	return base64.StdEncoding.EncodeToString(prvKey.Bytes())
}

func NewNodeKey() string {
	return NewPrivKey()
}

func ConvertJsonToPeers(jsonStr string) PeerList {
	var peers []Peer
	err := json.Unmarshal([]byte(jsonStr), &peers)
	if err != nil {
		log.Println("Error while parsing json string", err)
	}
	return peers
}

func GetNodeNameByNodeId(nodeId string, peers string) string {
	peerList := ConvertJsonToPeers(peers)
	if peer := peerList.GetPeerByNodeId(nodeId); peer != nil {
		return peer.NodeName
	}
	return ""
}

func GetNodeByNodeId(nodeId string, peers string) *Peer {
	peerList := ConvertJsonToPeers(peers)
	return peerList.GetPeerByNodeId(nodeId)
}

func ValidNodeArray(peers PeerList) error {

	for _, p := range peers {

		if p.NodeId == "" ||
			p.NodeKey == "" ||
			p.NodeName == "" ||
			p.PrivKey == "" ||
			p.NodeAddress == "" {
			return errors.New("nodeID or NodeKey or NodeName or PrivKey or NodeAddress is empty")
		}

		CorrectP2PPortIfNeedTo(&p.P2PPort)
		CorrectRPCPortIfNeedTo(&p.RPCPort)
		CorrectABCIPortIfNeedTo(&p.ABCIPort)
		CorrectWebServicePortIfNeedTo(&p.WebServicePort)
	}
	return nil
}
