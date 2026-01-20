package server

import (
	"chain/lib"
	"chain/rpc/common"
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
	"time"

	cmtjson "github.com/cometbft/cometbft/v2/libs/json"
)

func MakeLocalNet_Peer_03_04() lib.PeerList {
	peers := []lib.Peer{
		{
			NodeName:       "node03",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
			P2PPort:        46656,
			RPCPort:        46657,
			ABCIPort:       46658,
			WebServicePort: 3000,
			IsValidator:    true,
		},
		{
			NodeName:       "node04",
			NodeId:         lib.NewNodeId(),
			PrivKey:        lib.NewPrivKey(),
			NodeKey:        lib.NewNodeKey(),
			NodeAddress:    "127.0.0.1",
			P2PPort:        56656,
			RPCPort:        56657,
			ABCIPort:       56658,
			WebServicePort: 4000,
			IsValidator:    true,
		},
	}

	return peers
}

func TestSetupService_BroadcaseAddValidator(t *testing.T) {

	s := &SetupService{
		chainState: LoadChainState(chainStateFile),
	}
	s.chainState.InitializedTime = time.Now().UnixNano()
	s.chainState.GenesisDoc = lib.NewGenesisDoc("abc", nil)

	peers := MakeLocalNet_Peer_03_04()

	v := peers[0].BuildValidator()

	arg := common.SetupAddValidator{
		AddValidatorReq: lib.AddValidatorReq{
			PubType: v.PubKey.Type(),
			PubKey:  base64.StdEncoding.EncodeToString(v.PubKey.Bytes()),
			Power:   v.Power,
			Name:    v.Name,
		},
	}

	jsonBytes, err := cmtjson.Marshal(arg)

	fmt.Println("marshal:", string(jsonBytes), err)

	arg1 := common.SetupAddValidator{}
	err = cmtjson.Unmarshal([]byte(jsonBytes), &arg1)
	fmt.Println("unmarshal:", arg1, err)

	fmt.Println(reflect.DeepEqual(arg, arg1))

}
