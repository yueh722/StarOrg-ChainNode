package lib

import (
	"chain/utils"
	"fmt"

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

	url := fmt.Sprintf("%s/status", rpcAddr)

	_, err := utils.HttpGet(url)
	if err != nil {
		return false
	}

	return true
}

type StatusTransaction struct {
	Data   string `json:"data,omitempty"`
	Hash   string `json:"hash,omitempty"`
	Height int64  `json:"height,omitempty"`
}

// Tx
// Parameters
// hash (string): The hash of the transaction
// prove (bool): If the response should include proof the transaction was included in a block.

// Request
// Http
//
//	curl  http://127.0.0.1:26657/tx
//
// JSONRPC
//
//	curl -X POST https://localhost:26657 -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tx\"}"
func GetTransaction2(rpcAddr string, height int) ([]StatusTransaction, int, error) {

	url := fmt.Sprintf("%s/tx?hash=%s&prove=false", rpcAddr, "")

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

	if v := gjson.Parse(string(resultBytes)).Get("error\\.data").String(); v != "" {
		resultMap["error"] = v
		return resultMap
	}

	if v := gjson.Parse(string(resultBytes)).Get("result\\.hash").String(); v != "" {
		resultMap["hash"] = v
	}

	return resultMap
}
