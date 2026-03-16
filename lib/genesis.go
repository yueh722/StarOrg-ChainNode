package lib

import (
	"chain/lib/zlog"
	"chain/utils"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/v2/crypto/ed25519"
	"github.com/cometbft/cometbft/v2/crypto/secp256k1"
	"github.com/cometbft/cometbft/v2/types"
)

type AddValidatorReq struct {
	PubType string `json:"pub_type" binding:"required"`
	PubKey  string `json:"pubkey" binding:"required"` // base64 编码的字符串
	Power   int64  `json:"power" binding:"required"`
	Name    string `json:"name" binding:"required"`
}

type RemoveValidatorReq struct {
	PubType string `json:"pub_type" binding:"required"`
	PubKey  string `json:"pubkey" binding:"required"` // base64 编码的字符串
	Power   int64  `json:"power" binding:"required"`
	Name    string `json:"name" binding:"required"`
}

var pubKeyNameMap = map[string]string{
	types.ABCIPubKeyTypeEd25519:   ed25519.PubKeyName,
	types.ABCIPubKeyTypeSecp256k1: secp256k1.PubKeyName,
}

func ABCIPubKeyTypeToPubKeyName(pubKeyName string) (string, bool) {
	if v, ok := pubKeyNameMap[pubKeyName]; ok {
		return v, true
	}
	return "", false
}

func NewGenesisDoc(chainID string, validators []types.GenesisValidator) *types.GenesisDoc {
	return &types.GenesisDoc{
		ChainID:         "chain-" + chainID,
		ConsensusParams: types.DefaultConsensusParams(),
		GenesisTime:     GenesisTimeFromUnix(time.Now().UnixNano()),
		InitialHeight:   0,
		Validators:      validators,
	}
}

func SendAddValidatorRequest(webServiceAddr string, req *AddValidatorReq) error {

	if webServiceAddr == "" {
		return fmt.Errorf("web service address is empty")
	}

	zlog.Infof("sending add validator request to %v with nodeName: %v", webServiceAddr, req.Name)

	rawUrl := fmt.Sprintf("%s/validator/add", webServiceAddr)
	result, err := utils.HttpPost(rawUrl, req)
	if err != nil {
		return fmt.Errorf("send add validator request failure: %v", err)
	}

	zlog.Info("send add validator request success:", result)

	return nil
}

func SendRemoveValidatorRequest(webServiceAddr string, req *RemoveValidatorReq) error {

	if webServiceAddr == "" {
		return fmt.Errorf("web service address is empty")
	}

	zlog.Infof("sending remove validator request to %v with nodeName: %v", webServiceAddr, req.Name)

	rawUrl := fmt.Sprintf("%s/validator/remove", webServiceAddr)
	result, err := utils.HttpPost(rawUrl, req)
	if err != nil {
		return fmt.Errorf("send remove validator request failure: %v", err)
	}

	zlog.Info("send remove validator request success:", result)

	return nil
}
