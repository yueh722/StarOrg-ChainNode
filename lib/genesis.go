package lib

import (
	"time"

	"github.com/cometbft/cometbft/v2/types"
)

func NewGenesisDoc(chainID string, validators []types.GenesisValidator) *types.GenesisDoc {
	return &types.GenesisDoc{
		ChainID:         "chain-" + chainID,
		ConsensusParams: types.DefaultConsensusParams(),
		GenesisTime:     GenesisTimeFromUnix(time.Now().Unix()),
		InitialHeight:   0,
		Validators:      validators,
	}
}
