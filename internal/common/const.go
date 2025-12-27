package common

import (
	"chain/utils"
)

var (
	NodeMasterLogPath = utils.GetLogPath("NodeMaster")
	NodeProxyLogPath  = utils.GetLogPath("NodeProxy")

	NodeMasterAppDataPath = utils.GetAppDataPath("NodeMaster")
	NodeProxyAppDataPath  = utils.GetAppDataPath("NodeProxy")
)
