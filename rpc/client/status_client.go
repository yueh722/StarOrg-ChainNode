package client

import (
	"chain/lib"
	"chain/rpc/common"
	"chain/utils"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shirou/gopsutil/disk"
)

type StatusClient struct {
	client *rpc.Client
}

func newStatusClient(client *rpc.Client) *StatusClient {
	return &StatusClient{client: client}
}

func (c *StatusClient) Update(client *rpc.Client) {
	c.client = client
}

func (c *StatusClient) ReportNodeStatus() error {

	//load information of current node
	nodeServer, err := lib.GetNodeServer("")
	if err != nil {
		return err
	}

	err = c.client.Call(nil, "status_reportNodeStatus", common.StatusNode{
		NodeId:      "",
		NodeName:    nodeServer.NodeName,
		NodeAddress: "",
		Validator:   true,
		ABCIAddress: "",
		RPCAddress:  nodeServer.RPCAddr,
		Healthy:     true,
		LastCheck:   0,
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
