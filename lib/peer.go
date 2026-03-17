package lib

import (
	log "chain/lib/zlog"
	"chain/utils"
	"encoding/base64"
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
	"github.com/duke-git/lancet/v2/slice"
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
	Token          string `json:"token"`
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

func (p *Peer) RPCListenAddr() string {

	if p.RPCPort == 0 {
		return ""
	}
	// please note that cometbft node will listen rpc port which is always 127.0.0.1
	// ref: config.toml -> [rpc].laddr
	// default: laddr = "tcp://127.0.0.1:26657"
	return fmt.Sprintf("tcp://%s:%d", "127.0.0.1", p.RPCPort)
}

func (p *Peer) RPCAddr() string {
	if p.RPCPort == 0 {
		return ""
	}
	// please pay attention to listen address of RPCListenAddr before change the server address below.
	return fmt.Sprintf("http://%s:%d", "127.0.0.1", p.RPCPort)
}

func (p *Peer) RPCWebsocketAddrs() []string {
	return []string{
		fmt.Sprintf("ws://%s:%d/websocket", "127.0.0.1", p.RPCPort),
		fmt.Sprintf("wss://%s:%d/websocket", "127.0.0.1", p.RPCPort),
	}
}

func (p *Peer) ProxyAppAddr() string {
	if p.ABCIPort == 0 {
		return ""
	}
	return fmt.Sprintf("tcp://%s:%d", "127.0.0.1", p.ABCIPort)
}

func (p *Peer) WebServiceAddr() string {
	if p.WebServicePort == 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", p.NodeAddress, p.WebServicePort)
}

func (p *Peer) WebServiceListenAddr() string {
	if p.WebServicePort == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", "0.0.0.0", p.WebServicePort)
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

	prvKey, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		log.Fatalf("error decoding node key: %v", err)
	}

	return &p2p.NodeKey{
		PrivKey: ed25519.PrivKey(prvKey),
	}
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

	resErr := NewMultiError()

	//check if node name of peers are conflict.

	uniquePeers, err := slice.UniqueByField(peers, "NodeName")
	if err != nil {
		return err
	}

	diff := slice.Difference(peers, uniquePeers)
	if len(diff) > 0 {
		duplicatedNodeNames := slice.Map(diff, func(_ int, p Peer) string {
			return p.NodeName
		})
		return fmt.Errorf("node name [%v] had been found duplicated", strings.Join(duplicatedNodeNames, ","))
	}

	for i := range peers {
		p := &peers[i]

		//lookup domain or host, return error if failure
		if _, err := utils.CheckIpOrDomain(p.NodeAddress); err != nil {
			resErr.AddError(err)
		}

		if p.NodeId == "" ||
			p.NodeKey == "" ||
			p.NodeName == "" ||
			p.PrivKey == "" ||
			p.NodeAddress == "" {
			resErr.AddError(errors.New("nodeID or NodeKey or NodeName or PrivKey or NodeAddress is empty"))
		}

		CorrectP2PPortIfNeedTo(&p.P2PPort)
		CorrectRPCPortIfNeedTo(&p.RPCPort)
		CorrectABCIPortIfNeedTo(&p.ABCIPort)
		CorrectWebServicePortIfNeedTo(&p.WebServicePort)
	}

	return resErr.ErrorOrNil()
}
