package client

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/common"
	"chain/rpc/common/notification"
	"chain/utils"
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type LiveTxClient struct {
	client        *rpc.Client
	cfg           *ClientConfig
	conn          *websocket.Conn
	mux           sync.Mutex
	handleOffline func()
}

type ReportFunc func(*LiveTxClient, *lib.StatusTransaction) error

func newLiveTxClient(client *rpc.Client, cfg *ClientConfig) *LiveTxClient {

	c := &LiveTxClient{
		client: client,
		cfg:    cfg,
	}

	ConnNotificationMgr().Register(c)

	go c.StartLiveTxTransmitter()

	return c
}

func (c *LiveTxClient) OnNotify(msg *notification.Message) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if msg.Type == notification.MsgTypeOffline {
		zlog.Info("got notification of server offline: ", msg.Payload.(string))
		if c.handleOffline != nil {
			c.handleOffline()
		}
	}
}

func (c *LiveTxClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *LiveTxClient) ReportLiveTx(tx *lib.StatusTransaction) error {
	nodeInfo := *c.cfg.NodeInfo

	//send tx to master
	err := c.client.Call(nil, common.GetMethodName("ReportLiveTx"), nodeInfo.NodeID, tx)
	if err != nil {
		zlog.Info("failed to report live tx to master, ", err)
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

func ListenTransaction(ctx context.Context, conn *websocket.Conn) <-chan lib.StatusTransaction {
	txDetailChan := make(chan lib.StatusTransaction, 100)

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
				zlog.Info("[live transaction] failed to read message from tx subscription:", err)
				break
			}

			txBase64 := gjson.Parse(string(message)).Get("result.data.value.TxResult.tx").String()
			//txBytes, _ := base64.StdEncoding.DecodeString(txBase64)

			height := gjson.Parse(string(message)).Get("result.data.value.TxResult.height").Int()
			txHash := gjson.Parse(string(message)).Get(`result.events.tx\.hash.0`).String()

			if txBase64 == "" || txHash == "" {
				continue
			}

			//zlog.Infof("[live transaction] txBase64 bytes: %v, height: %v, hash: %v", txBase64, height, (txHash))

			select {
			case <-ctx.Done():
				return
			case txDetailChan <- lib.StatusTransaction{
				Data:   txBase64,
				Hash:   txHash,
				Height: height,
			}:
			}

		}
	}()

	return txDetailChan
}

func (c *LiveTxClient) CheckEnabledTransmitter() bool {
	nodeInfo := *c.cfg.NodeInfo
	var enableTransmitter bool
	err := c.client.Call(&enableTransmitter, common.GetMethodName("EnableTransmitter"),
		nodeInfo.NodeID,
		lib.CheckNodeOnline(nodeInfo.RPCAddr))
	if err != nil {
		zlog.Info("[live transaction] failed to check node enabled transmitter ", err)
	}

	return enableTransmitter
}

func (c *LiveTxClient) StartLiveTxTransmitter() {

	logger := utils.NewThrottledLogger(zlog.Info)

	for {
		nodeInfo := *c.cfg.NodeInfo

		//check if current peer is online
		if lib.CheckNodeOnline(nodeInfo.RPCAddr) == false {

			if c.cfg.Initialized {
				logger.Log("[live transaction] node rpc is offline: ", nodeInfo.NodeName, nodeInfo.RPCAddr)
			}

			time.Sleep(3 * time.Second)
			continue
		}

		//let master tell that if we need to read live tx from rpc of node.
		if !c.CheckEnabledTransmitter() {
			time.Sleep(3 * time.Second)
			continue
		}

		wsconn := mustGetConnection(lib.RPCWebsocketAddrs(&nodeInfo))

		zlog.Info("[live transaction] connected to node RPC: ", nodeInfo.NodeName, nodeInfo.RPCAddr)
		ctx, cancelReadingFn := context.WithCancel(context.Background())
		txCh := ListenTransaction(ctx, wsconn)

		c.handleOffline = func() {
			cancelReadingFn()
			_ = wsconn.Close()
			zlog.Infof("[live transaction] handled server offline, cancel reading tx, close node rpc connection.")
		}

		for tx := range txCh {
			//report tx
			if err := c.ReportLiveTx(&tx); err != nil {
				break
			}
		}

	}

}
