package client

import (
	"chain/rpc/common"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shirou/gopsutil/disk"
)

type StatusClient struct {
	client *rpc.Client
	cfg    *ClientConfig
}

func newStatusClient(client *rpc.Client, cfg *ClientConfig) *StatusClient {
	return &StatusClient{
		client: client,
		cfg:    cfg,
	}
}

func (c *StatusClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *StatusClient) ReportNodeStatus() error {

	//load information of current node
	statusNode, err := BuildStatusNode(c.cfg, nil)
	if err != nil {
		return err
	}

	err = c.client.Call(nil, "status_reportNodeStatus", statusNode)
	return err
}

func (c *StatusClient) ReportStorageStatus() error {

	info, _ := disk.Usage(c.cfg.NodeBaseDir[:len("c:/")])

	err := c.client.Call(nil, "status_reportStorageStatus", common.StatusStorage{
		Path:        info.Path,
		TotalSpace:  info.Total,
		FreeSpace:   info.Free,
		UsedSpace:   info.Used,
		UsedPercent: info.UsedPercent,
	})

	return err
}
