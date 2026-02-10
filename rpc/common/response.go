package common

import (
	"fmt"
)

type RpcResponse struct {
	Message string `json:"message"`
}

func MakeRpcSucceed() (*RpcResponse, error) {
	return &RpcResponse{Message: "succeed"}, nil
}

func MakeRpcErrorf(format string, a ...any) (*RpcResponse, error) {
	return nil, fmt.Errorf(format, a...)
}

func MakeRpcError(err error) (*RpcResponse, error) {
	return nil, err
}
