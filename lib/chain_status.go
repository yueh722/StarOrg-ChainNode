package lib

import (
	"chain/utils"
	"encoding/json"
	"fmt"

	coretypes "github.com/cometbft/cometbft/v2/rpc/core/types"
	"github.com/tidwall/gjson"
)

const (
	RpcStatus = "status"
)

var rpcURLs = map[string]string{
	RpcStatus: "%s/status",
}

func rpcRoutine(name string) any {
	return func(node *NodeInfo) {
		switch name {
		case RpcStatus:

		}
	}
}

func CheckNodeOnline(rpcAddr string) bool {

	url := fmt.Sprintf("%s/v2/status", rpcAddr)

	_, err := utils.HttpGet(url)
	if err != nil {
		return false
	}

	return true
}

func GetActivatedNode(peers PeerList) *Peer {
	if peers == nil {
		return nil
	}

	for _, peer := range peers {
		if CheckNodeOnline(peer.ToRPCAddr()) == true {
			return &peer
		}
	}

	return nil
}

type TransactionDetail struct {
	Key   string
	Value string
}

// Tx
// Parameters
// hash (string): The hash of the transaction
// prove (bool): If the response should include proof the transaction was included in a block.

// Request
// Http
//
//	curl  http://127.0.0.1:26657/v1/tx
//
// JSONRPC
//
//	curl -X POST https://localhost:26657/v1 -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tx\"}"
func GetTransaction2(rpcAddr string, height int) ([]TransactionDetail, int, error) {

	url := fmt.Sprintf("%s/v2/tx?hash=%s&prove=false", rpcAddr, "")

	_, err := utils.HttpGet(url)
	if err != nil {
		return nil, 0, err
	}

	return nil, 0, nil
}

func makeResultData(resultBytes []byte) map[string]interface{} {

	resultMap := map[string]interface{}{
		"hash": nil,
	}

	if resultBytes == nil {
		return resultMap
	}

	if v := gjson.Parse(string(resultBytes)).Get("error.data").String(); v != "" {
		resultMap["error"] = v
		return resultMap
	}

	if v := gjson.Parse(string(resultBytes)).Get("result.hash").String(); v != "" {
		resultMap["hash"] = v
	}

	return resultMap
}

// block_results
// height (integer): Height of the block which contains the results. If no height is specified, the latest block height will be used
func GetTransaction(rpcAddr string, height *int64) ([]TransactionDetail, error) {
	url := fmt.Sprintf("%s/v2/block_results", rpcAddr)
	if *height > 0 {
		url = fmt.Sprintf("%s/v2/block_results?height=%v", rpcAddr, height)
	}
	rawRes, err := utils.HttpGet(url)
	if err != nil {
		return nil, err
	}

	blockResult := &coretypes.ResultBlockResults{}
	if err := json.Unmarshal(rawRes.([]byte), blockResult); err != nil {
		return nil, err
	}

	*height = blockResult.Height

	txDetails := []TransactionDetail{}

	for _, txRes := range blockResult.TxResults {
		for _, event := range txRes.Events {
			for _, attribute := range event.Attributes {
				txDetails = append(txDetails, TransactionDetail{
					Key:   attribute.GetKey(),
					Value: attribute.GetValue(),
				})
			}
		}
	}

	return txDetails, nil
}
