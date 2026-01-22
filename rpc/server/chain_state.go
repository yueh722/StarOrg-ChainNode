package server

import (
	"chain/lib"
	"chain/rpc/common"
	"fmt"
	"os"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
	"github.com/cometbft/cometbft/v2/types"
)

type ChainStateInternal struct {
	Peers           lib.PeerList      `json:"peers"`
	GenesisDoc      *types.GenesisDoc `json:"genesis_doc"`
	InitializedTime int64             `json:"initialized_time"`
	LastAddNodeTime int64             `json:"last_add_node_time"`
}

type ChainState struct {
	Nodes           []common.StatusNode `json:"nodes"`
	ChainID         string              `json:"chain_id"`
	InitializedTime int64               `json:"initialized_time"`
	LastAddNodeTime int64               `json:"last_add_node_time"`
}

func LoadChainState(filePath string) (*ChainStateInternal, error) {

	state := &ChainStateInternal{}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error reading chain state file: %v", err)
	}

	err = cmtjson.Unmarshal(file, state)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling file contents to chain state: %v", err)
	}

	return state, nil
}

func (state *ChainStateInternal) Initialized() bool {
	return state.GenesisDoc != nil && state.InitializedTime != 0
}

func (state *ChainStateInternal) IsNodeInitialized(nodeID string) bool {
	for _, node := range state.Peers {
		if node.NodeId == nodeID {
			return true
		}
	}

	return false
}

func NewChainState(chainID string, peers lib.PeerList) *ChainStateInternal {

	GenesisDoc := lib.NewGenesisDoc(chainID, peers.BuildValidators())

	return &ChainStateInternal{
		Peers:           peers,
		GenesisDoc:      GenesisDoc,
		InitializedTime: time.Now().UnixNano(),
	}
}

func (state *ChainStateInternal) Clone() *ChainStateInternal {

	newGenesisDoc := *state.GenesisDoc
	newGenesisDoc.Validators = append([]types.GenesisValidator(nil), state.GenesisDoc.Validators...)

	newState := &ChainStateInternal{
		Peers:           append(lib.PeerList{}, state.Peers...),
		GenesisDoc:      &newGenesisDoc,
		InitializedTime: state.InitializedTime,
		LastAddNodeTime: state.LastAddNodeTime,
	}

	return newState
}

func (state *ChainStateInternal) CloneAddNode(peerInfo lib.Peer) *ChainStateInternal {

	newPeers := append(state.Peers, peerInfo)

	newState := &ChainStateInternal{
		Peers:           newPeers,
		GenesisDoc:      state.GenesisDoc,
		InitializedTime: state.InitializedTime,
		LastAddNodeTime: time.Now().UnixNano(),
	}

	return newState
}

func (s *ChainStateInternal) Save(filePath string) error {
	data, err := cmtjson.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *ChainStateInternal) Reset(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove chain state file %s, err: %v", filePath, err)
	}

	*s = ChainStateInternal{}
	return nil
}
