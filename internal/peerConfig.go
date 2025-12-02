package internal

import (
	"github.com/cometbft/cometbft/v2/config"
	"github.com/cometbft/cometbft/v2/crypto"
	"github.com/cometbft/cometbft/v2/types"
	"time"
)

type PeerConfig struct {
	config  *config.Config
	genesis types.GenesisDoc
	privKey crypto.PrivKey
	nodeId  string
}

func (cfg *PeerConfig) UpdateConfig(node *Peer) error {
	return nil
}

func (cfg *PeerConfig) NewGensis(chainId string, genesisTime time.Time) error {
	return nil
}

func (cfg *PeerConfig) SetGensis(genesis *types.GenesisDoc) {
	cfg.genesis = *genesis
}

func (cfg *PeerConfig) SaveFiles() error {
	return nil
}
