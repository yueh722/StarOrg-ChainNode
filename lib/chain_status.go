package lib

import (
	"chain/lib/zlog"
	"chain/utils"
	"fmt"
	"time"
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

func GetActivatedNode(peers PeerList) *Peer {
	if peers == nil {
		return nil
	}

	for _, peer := range peers {
		if CheckNodeOnline(peer.ToRPCAddr()) == true {
			return &peer
		}
	}

	return nil
}

func GetTransaction(rpcAddr string, height int) ([]TransactionDetail, int, error) {

	return nil, 0, nil
}

type TransactionDetail struct {
}

func ListenTransaction(peer *Peer) <-chan TransactionDetail {
	txDetailChan := make(chan TransactionDetail)

	go func() {
		defer func() {
			close(txDetailChan)
		}()
		sinceWeight := 0
		for {
			if CheckNodeOnline(peer.ToRPCAddr()) == false {
				return
			}

			txs, weight, err := GetTransaction(peer.ToRPCAddr(), sinceWeight)
			if err != nil {
				zlog.Info("GetTransaction err:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			sinceWeight += weight
			for _, tx := range txs {
				txDetailChan <- tx
			}
		}
	}()

	return txDetailChan
}
