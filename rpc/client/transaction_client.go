package client

import (
	"chain/lib"
	"chain/rpc/common"

	"github.com/ethereum/go-ethereum/rpc"
)

type LiveTxClient struct {
	client *rpc.Client
	cfg    *ClientConfig
}

func newLiveTxClient(client *rpc.Client, cfg *ClientConfig) *LiveTxClient {

	c := &LiveTxClient{
		client: client,
		cfg:    cfg,
	}

	go c.StartTransmit()

	return c
}

func (c *LiveTxClient) StartTransmit() {

}

func (c *LiveTxClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *LiveTxClient) ReportLiveTx() error {

	//read live transaction

	//send tx to master

	method := common.MakeMethodName(common.StreamNamespace{}.GetNamesapceString(), "ReportState")

	err := c.client.Call(nil, method, lib.StatusTransaction{})
	return err
}
