package callserver

import (
	"chain/rpc/common"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcServer(client *rpc.Client, method string, arg interface{}, reply interface{}) error {
	return client.Call(reply, common.GetMethodName(method), arg)
}
