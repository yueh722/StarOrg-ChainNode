package client

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/common/namespace"
	"chain/rpc/common/setupTypes"
	"chain/utils"
	"fmt"
	"os"
	"path/filepath"

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
	if err := c.client.RegisterName(namespace.StatusClientNamespace{}.GetNamesapceString(), c); err != nil {
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

	if !c.cfg.Initialized {
		zlog.Info("[IsNodeRpcAlive] node is not initialized yet, deny")
		return false
	}

	ret := lib.CheckNodeOnline(c.cfg.NodeInfo.RPCAddr)
	zlog.Info("[IsNodeRpcAlive] checking node rpc:", c.cfg.NodeInfo.RPCAddr, ret)
	return ret
}

func (c *StatusClient) ReportNodeStatus() error {

	//load information of current node
	statusNode, err := BuildStatusNode(c.cfg, nil)
	if err != nil {
		return err
	}

	err = c.client.Call(nil, namespace.GetMethodName("ReportNodeStatus"), statusNode)
	return err
}

func (c *StatusClient) ReportStorageStatus() error {

	nodeID := ""
	nodeName := ""
	if c.cfg != nil && c.cfg.NodeInfo != nil {
		nodeID = c.cfg.NodeInfo.NodeID
		nodeName = c.cfg.NodeInfo.NodeName
	}

	hostname, _ := os.Hostname()

	drive := filepath.VolumeName(c.cfg.HomeDir)
	if drive != "" {
		drive += string(filepath.Separator)
	} else if len(c.cfg.NodeBaseDir) >= 3 {
		drive = c.cfg.NodeBaseDir[:len("A:/")]
	} else {
		drive = "/"
	}

	info, err := disk.Usage(drive)
	if err != nil {
		info, err = disk.Usage(c.cfg.HomeDir)
		if err != nil {
			zlog.Errorf("failed to get disk usage, err: %v", err)
			return nil
		}
	}

	// 鏈資料庫空間 (data 資料庫 + app 身分認證資料庫)
	nodeBaseDir := c.cfg.NodeBaseDir
	dataPath := filepath.Join(nodeBaseDir, "data")
	appPath := filepath.Join(nodeBaseDir, "app")
	chainDataBytes := utils.GetDirectorySize(dataPath) + utils.GetDirectorySize(appPath)

	// 該節點整體系統目錄佔用大小 (HomeDir)
	totalNodeBytes := utils.GetDirectorySize(c.cfg.HomeDir)
	if totalNodeBytes == 0 && nodeBaseDir != "" {
		totalNodeBytes = utils.GetDirectorySize(nodeBaseDir)
	}

	status := setupTypes.StatusStorage{
		NodeID:             nodeID,
		NodeName:           nodeName,
		HostName:           hostname,
		Path:               info.Path,
		TotalSpace:         utils.ConvertBytesToReadable(info.Total),
		FreeSpace:          utils.ConvertBytesToReadable(info.Free),
		UsedSpace:          utils.ConvertBytesToReadable(info.Used),
		UsedPercent:        fmt.Sprintf("%.1f%%", info.UsedPercent),
		DiskTotalBytes:     info.Total,
		DiskFreeBytes:      info.Free,
		ChainDataSize:      utils.ConvertBytesToReadable(chainDataBytes),
		TotalChainnodeSize: utils.ConvertBytesToReadable(totalNodeBytes),
		DataSizeBytes:      chainDataBytes,
		TotalSizeBytes:     totalNodeBytes,
		UpdatedAt:          utils.NowString(),
	}

	err = c.client.Call(nil, namespace.GetMethodName("ReportStorageStatus"), status)
	return err
}
