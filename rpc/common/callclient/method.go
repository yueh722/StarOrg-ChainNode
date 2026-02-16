package callclient

import (
	"chain/lib/zlog"
	"chain/rpc/common/namespace"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcClient(reply interface{}, client *rpc.Client, method string, args ...interface{}) error {
	return client.Call(reply, namespace.GetMethodName(method), args...)
}

func IsNodeRpcAlive(client *rpc.Client) bool {
	var rpcAlive bool
	err := CallRpcClient(&rpcAlive, client, "IsNodeRpcAlive")
	if err != nil {
		zlog.Info("IsNodeRpcAlive err:", err)
		return false
	}
	return rpcAlive
}
