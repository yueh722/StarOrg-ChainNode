package client

import (
	"chain/lib"
	"chain/rpc/common"
	"chain/utils"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shirou/gopsutil/disk"
)

type StatusClient struct {
	client *rpc.Client
	cfg    ClientConfig
}

func newStatusClient(client *rpc.Client, cfg ClientConfig) *StatusClient {
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
	nodeInfo, err := lib.GetNodeInfo(c.cfg.HomeDir)
	if err != nil {
		return err
	}

	//call RPCAddr/status to check healthy

	err = c.client.Call(nil, "status_reportNodeStatus", common.StatusNode{
		NodeInfo:    *nodeInfo,
		Healthy:     lib.CheckNodeOnline(nodeInfo.RPCAddr),
		LastCheck:   time.Now().UnixNano(),
		StartedTime: utils.GetProcessCreatedTime(lib.NodeFileName, lib.AppFileName),
	})
	return err
}

func (c *StatusClient) ReportStorageStatus() error {

	info, _ := disk.Usage(utils.GetStorageDrive())

	err := c.client.Call(nil, "status_reportStorageStatus", common.StatusStorage{
		Path:        info.Path,
		TotalSpace:  info.Total,
		FreeSpace:   info.Free,
		UsedSpace:   info.Used,
		UsedPercent: info.UsedPercent,
	})

	return err
}
