package callclient

import (
	"chain/lib/zlog"
	"chain/rpc/common"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcClient(client *rpc.Client, method string, arg interface{}, reply interface{}) error {
	return client.Call(reply, common.GetMethodName(method), arg)
}

func IsNodeRpcAlive(client *rpc.Client) bool {
	var rpcAlive bool
	err := CallRpcClient(client, "IsNodeRpcAlive", nil, &rpcAlive)
	if err != nil {
		zlog.Info("IsNodeRpcAlive err:", err)
		return false
	}
	return rpcAlive
}
