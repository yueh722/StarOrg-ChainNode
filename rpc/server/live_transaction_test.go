package server

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type WSResult struct {
	Query string `json:"query"`
	Data  struct {
		Type  string `json:"type"`
		Value struct {
			TxResult struct {
				Height string      `json:"height"`
				Index  int32       `json:"index"`
				Tx     string      `json:"tx"`
				Result interface{} `json:"result"` // 空对象所以用 interface
			} `json:"TxResult"`
		} `json:"value"`
	} `json:"data"`

	Events map[string][]string `json:"events"`
}

func TestTxEvent(t *testing.T) {
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:26657/websocket", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 订阅 Tx
	msg := `{"jsonrpc":"2.0","method":"subscribe","id":"1","params":{"query":"tm.event='Tx'"}}`
	_ = c.WriteMessage(websocket.TextMessage, []byte(msg))

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}

		//fmt.Println("WS RAW:", string(message))

		/*
			resJson := `{
				  "jsonrpc": "2.0",
				  "id": "1",
				  "result": {
				    "query": "tm.event='Tx'",
				    "data": {
				      "type": "tendermint/event/Tx",
				      "value": {
				        "TxResult": {
				          "height": "35",
				          "index": 0,
				          "tx": "base64 string",
				          "result": {}
				        }
				      }
				    },
				    "events": {
				      "tx.height": [
				        "35"
				      ],
				      "tm.event": [
				        "Tx"
				      ],
				      "tx.hash": [
				        "hash1"
				      ]
				    }
				  }
				}`
		*/

		txBase64 := gjson.Parse(string(message)).Get("result.data.value.TxResult.tx").String()
		txBytes, _ := base64.StdEncoding.DecodeString(txBase64)

		// get key: {"a.b": 111}
		// `a\.b`
		// "a\\.b"
		txHash := gjson.Parse(string(message)).Get(`result.events.tx\.hash.0`).String()

		fmt.Println("decoded tx bytes:", string(txBytes), "hash:", string(txHash))
	}
}
