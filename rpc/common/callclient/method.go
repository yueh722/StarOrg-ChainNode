package callclient

import (
	"chain/lib/zlog"
	"chain/rpc/common/namespace"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcClient(reply interface{}, conn *rpc.Client, method string, args ...interface{}) error {
	return conn.Call(reply, namespace.GetMethodName(method), args...)
}

func IsNodeRpcAlive(conn *rpc.Client) bool {
	var rpcAlive bool
	err := CallRpcClient(&rpcAlive, conn, "IsNodeRpcAlive")
	if err != nil {
		zlog.Info("IsNodeRpcAlive err:", err)
		return false
	}
	return rpcAlive
}
