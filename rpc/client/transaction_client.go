package client

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/common"
	"chain/utils"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type LiveTxClient struct {
	client *rpc.Client
	cfg    *ClientConfig
	conn   *websocket.Conn
	mux    sync.Mutex
}

type ReportFunc func(*LiveTxClient, *lib.StatusTransaction) error

func newLiveTxClient(client *rpc.Client, cfg *ClientConfig) *LiveTxClient {

	c := &LiveTxClient{
		client: client,
		cfg:    cfg,
	}

	if err := client.RegisterName(common.LiveTxNamespace{}.GetNamesapceString(), c); err != nil {
		zlog.Infof("Failed to register liveTx service: %v", err)
		return nil
	}

	//go c.StartLiveTxTransmitter()

	return c
}

func (c *LiveTxClient) StartLiveTxTransmitter() {
	//check if current peer is online
	//if true call method of Server to check if need to read live txs
	c.startTransmitLiveTransactionFromLocal()
}

func (c *LiveTxClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *LiveTxClient) ReportLiveTx(tx *lib.StatusTransaction) error {
	//send tx to master
	method := common.MakeMethodName(common.StreamNamespace{}.GetNamesapceString(), "ReportState")
	err := c.client.Call(nil, method, tx)
	if err != nil {
		zlog.Info("failed to report live tx to master", "err", err)
	}
	return err
}

func mustGetConnection(addresses []string) *websocket.Conn {
	for {
		for _, address := range addresses {

			zlog.Info("[live transaction] prepare connecting to ", address)
			c, _, err := websocket.DefaultDialer.Dial(address, nil)
			if err != nil {
				zlog.Error("[live transaction] failed to connect websocket of node RPC ", err)
				time.Sleep(3 * time.Second)
				continue
			}

			zlog.Info("[live transaction] successfully connected to node RPC: ", address)
			return c
		}

		time.Sleep(5 * time.Second)
	}
}

func ListenTransaction(conn *websocket.Conn) <-chan lib.StatusTransaction {
	txDetailChan := make(chan lib.StatusTransaction)

	go func() {
		defer func() {
			close(txDetailChan)
		}()

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

func (c *LiveTxClient) startTransmitLiveTransactionFromLocal() {

	logger := utils.NewThrottledLogger(zlog.Info)

	for {

		nodeInfo := *c.cfg.NodeInfo

		if lib.CheckNodeOnline(nodeInfo.RPCAddr) == false {
			logger.Log("[live transaction] node rpc is offline: ", nodeInfo.NodeName, nodeInfo.RPCAddr)
			time.Sleep(3 * time.Second)
			continue
		}

		//call method to check if we need to continue to transmit txs
		//enableTransmitter := method(nodeID, online)
		//if !enableTransmitter sleep 3, contine

		wsconn := mustGetConnection(lib.RPCWebsocketAddrs(&nodeInfo))
		go func(conn *websocket.Conn) {
			//check if we need to stop transmitter
			//if true
			//conn.Close()
		}(wsconn)

		logger.Log("[live transaction] peer is online: ", nodeInfo.NodeName, nodeInfo.RPCAddr)
		txCh := ListenTransaction(wsconn)
		for tx := range txCh {

			//report tx
			_ = c.ReportLiveTx(&tx)
			time.Sleep(10 * time.Millisecond)
		}
	}

}
