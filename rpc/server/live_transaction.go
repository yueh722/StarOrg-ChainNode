package server

import (
	"chain/lib"
	"chain/lib/zlog"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

func mustGetConnection(addresses []string) *websocket.Conn {
	for {
		for _, address := range addresses {

			zlog.Info("prepare connecting to ", address)
			c, _, err := websocket.DefaultDialer.Dial(address, nil)
			if err != nil {
				zlog.Error("failed to connect websocket of node RPC", err)
				continue
			}

			zlog.Info("successfully connected to node RPC:", address)
			return c
		}

		time.Sleep(5 * time.Second)
	}
}

func ListenTransaction(peer *lib.Peer) <-chan lib.StatusTransaction {
	txDetailChan := make(chan lib.StatusTransaction)

	go func() {
		defer func() {
			close(txDetailChan)
		}()

		if lib.CheckNodeOnline(peer.ToRPCAddr()) == false {
			return
		}

		conn := mustGetConnection(peer.ToWebsocketAddr())

		// subscribe Tx
		msg := `{"jsonrpc":"2.0","method":"subscribe","id":"1","params":{"query":"tm.event='Tx'"}}`
		_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				zlog.Error("failed to read message from tx subscription:", err)
				break
			}

			txBase64 := gjson.Parse(string(message)).Get("result.data.value.TxResult.tx").String()
			//txBytes, _ := base64.StdEncoding.DecodeString(txBase64)

			height := gjson.Parse(string(message)).Get("result.data.value.TxResult.height").Int()
			txHash := gjson.Parse(string(message)).Get(`result.events.tx\.hash.0`).String()

			if txBase64 == "" || txHash == "" {
				continue
			}

			zlog.Info("txBase64 bytes:", txBase64, "height:", height, "hash:", (txHash))

			txDetailChan <- lib.StatusTransaction{
				Data:   txBase64,
				Hash:   txHash,
				Height: height,
			}
		}
	}()

	return txDetailChan
}

func startTransmitLiveTransaction(stream *StreamService) {

	//read chain state
	chainStateInternal := LoadChainState(chainStateFile)

	// start go-routine to fetch live transactions from node and transmit tx to transaction subscribers
	go func() {

		for {
			activatedPeer := lib.GetActivatedNode(chainStateInternal.Peers)
			if activatedPeer == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			zlog.Println("found activated peer:", activatedPeer.NodeName, activatedPeer.ToRPCAddr())
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
