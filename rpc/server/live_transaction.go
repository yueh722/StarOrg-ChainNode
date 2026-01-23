package server

/*
// reading live tx from node RPC issued by server-side
func mustGetConnection(addresses []string) *websocket.Conn {
	for {
		for _, address := range addresses {

			zlog.Info("[live transaction] prepare connecting to ", address)
			c, _, err := websocket.DefaultDialer.Dial(address, nil)
			if err != nil {
				zlog.Error("[live transaction] failed to connect websocket of node RPC ", err)
				continue
			}

			zlog.Info("[live transaction] successfully connected to node RPC: ", address)
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

		conn := mustGetConnection(peer.RPCWebsocketAddrs())

		// subscribe Tx
		msg := `{"jsonrpc":"2.0","method":"subscribe","id":"1","params":{"query":"tm.event='Tx'"}}`
		_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				zlog.Error("[live transaction] failed to read message from tx subscription:", err)
				break
			}

			txBase64 := gjson.Parse(string(message)).Get("result.data.value.TxResult.tx").String()
			//txBytes, _ := base64.StdEncoding.DecodeString(txBase64)

			height := gjson.Parse(string(message)).Get("result.data.value.TxResult.height").Int()
			txHash := gjson.Parse(string(message)).Get(`result.events.tx\.hash.0`).String()

			if txBase64 == "" || txHash == "" {
				continue
			}

			zlog.Infof("[live transaction] txBase64 bytes: %v, height: %v, hash: %v", txBase64, height, (txHash))

			txDetailChan <- lib.StatusTransaction{
				Data:   txBase64,
				Hash:   txHash,
				Height: height,
			}
		}
	}()

	return txDetailChan
}

func startTransmitLiveTransactionFromServer(stream *StreamService) {

	// start go-routine to fetch live transactions from node and transmit tx to transaction subscribers
	go func() {

		logger := utils.NewThrottledLogger(zlog.Info)

		for {

			var (
				chainStateInternal = &ChainStateInternal{}
				err                error
			)

			if chainStateInternal.InitializedTime == 0 || chainStateInternal.Peers == nil {
				chainStateInternal, err = LoadChainState(chainStateFile)
				if err != nil {
					logger.Log("[live transaction] ", err)
					continue
				}
			}

			activatedPeer := lib.GetActivatedNode(chainStateInternal.Peers)
			if activatedPeer == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			zlog.Printf("[live transaction] found activated peer: %v %v", activatedPeer.NodeName, activatedPeer.RPCAddr())
			txCh := ListenTransaction(activatedPeer)
			for tx := range txCh {
				if ok := stream.ReportState(tx); !ok {
					zlog.Info("[live transaction] failed to transmit live transaction")
				}
				time.Sleep(1 * time.Second)
			}
		}

	}()

}

*/
