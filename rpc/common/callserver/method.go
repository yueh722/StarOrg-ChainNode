package callserver

import (
	"chain/rpc/common/namespace"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcServer(reply interface{}, client *rpc.Client, method string, args ...interface{}) error {
	return client.Call(reply, namespace.GetMethodName(method), args...)
}
