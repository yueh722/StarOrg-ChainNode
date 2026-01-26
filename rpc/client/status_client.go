package client

import (
	"chain/lib"
	"chain/rpc/common"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shirou/gopsutil/disk"
)

type StatusClient struct {
	client *rpc.Client
	cfg    *ClientConfig
}

func newStatusClient(client *rpc.Client, cfg *ClientConfig) *StatusClient {

	c := &StatusClient{
		client: client,
		cfg:    cfg,
	}

	c.registerService()

	return c
}

func (c *StatusClient) registerService() {
	if err := c.client.RegisterName(common.StatusClientNamespace{}.GetNamesapceString(), c); err != nil {
		panic(fmt.Errorf("failed to register StatusClient, err: %v", err))
	}
}

func (c *StatusClient) Update(client *rpc.Client) {
	c.client = client

	//It has to re-register service upon the connection established.
	//Because rpc methods are bind with each client. If u don't do this, the map of rpc methods will lose, which means u cannot call rpc method of client via client pointer.
	c.registerService()
}

func (c *StatusClient) IsNodeRpcAlive() bool {
	return lib.CheckNodeOnline(c.cfg.NodeInfo.RPCAddr)
}

func (c *StatusClient) ReportNodeStatus() error {

	//load information of current node
	statusNode, err := BuildStatusNode(c.cfg, nil)
	if err != nil {
		return err
	}

	err = c.client.Call(nil, common.GetMethodName("ReportNodeStatus"), statusNode)
	return err
}

func (c *StatusClient) ReportStorageStatus() error {

	info, _ := disk.Usage(c.cfg.NodeBaseDir[:len("c:/")])

	err := c.client.Call(nil, common.GetMethodName("ReportStorageStatus"), common.StatusStorage{
		Path:        info.Path,
		TotalSpace:  info.Total,
		FreeSpace:   info.Free,
		UsedSpace:   info.Used,
		UsedPercent: info.UsedPercent,
	})

	return err
}
