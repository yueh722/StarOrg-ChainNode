package server

// callback of node error
// func

// nodeproxy send error log of cometbft node to NodeMaster
// Request from: NodeProxy
type NodeErrorReceiverService struct {
	ErrResponseSrv LogService
}
