package callserver

import (
	"chain/lib/zlog"
	"chain/rpc/common/namespace"

	"github.com/ethereum/go-ethereum/rpc"
)

func CallRpcServer(reply interface{}, conn *rpc.Client, method string, args ...interface{}) error {
	return conn.Call(reply, namespace.GetMethodName(method), args...)
}

func IsNodeInitialized(conn *rpc.Client, nodeID string) bool {
	var initialized bool
	err := CallRpcServer(&initialized, conn, "GetNodeInitializedStatus", nodeID)
	if err != nil {
		zlog.Info("GetNodeInitializedStatus err:", err)
		return false
	}
	return initialized
}
