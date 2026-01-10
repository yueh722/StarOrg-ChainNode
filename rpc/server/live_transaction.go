package server

import (
	"chain/lib"
	"chain/lib/zlog"
	"sync"
	"time"
)

var once sync.Once

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

			txCh := lib.ListenTransaction(activatedPeer)
			for tx := range txCh {
				if ok := stream.ReportState(tx); !ok {
					zlog.Info("failed to transmit live transaction")
				}
				time.Sleep(1 * time.Second)
			}
		}

	}()

}
