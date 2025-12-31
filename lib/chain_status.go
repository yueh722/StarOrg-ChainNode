package lib

import (
	"chain/utils"
	"fmt"
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

func CheckNodeOnline(node *NodeInfo) bool {

	url := fmt.Sprintf("%s/status", node.RPCAddr)

	_, err := utils.HttpGet(url)
	if err != nil {
		return false
	}

	return true
}
