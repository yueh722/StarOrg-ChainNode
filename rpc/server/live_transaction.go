package server

import (
	"chain/lib"
	"chain/lib/zlog"
	"time"
)

func ListenTransaction(peer *lib.Peer) <-chan lib.TransactionDetail {
	txDetailChan := make(chan lib.TransactionDetail)

	go func() {
		defer func() {
			close(txDetailChan)
		}()

		var sinceHeight int64 = 0

		for {
			if lib.CheckNodeOnline(peer.ToRPCAddr()) == false {
				return
			}

			txs, err := lib.GetTransaction(peer.ToRPCAddr(), &sinceHeight)
			if err != nil {
				zlog.Info("GetTransaction err:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, tx := range txs {
				txDetailChan <- tx
			}
		}
	}()

	return txDetailChan
}

func transmitLiveTransaction(stream *StreamService) {

	//read chain state
	chainStateInternal := LoadChainState(chainStateFile)

	// start go-routine to fetch live transactions from node and transmit to transaction subscribers
	go func() {

		for {
			activatedPeer := lib.GetActivatedNode(chainStateInternal.Peers)
			if activatedPeer == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			txCh := ListenTransaction(activatedPeer)
			for tx := range txCh {
				if ok := stream.ReportState(tx); !ok {
					zlog.Info("failed to transmit live transaction")
				}
				time.Sleep(1 * time.Second)
			}
		}

	}()

}
