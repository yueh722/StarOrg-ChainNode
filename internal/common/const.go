package common

import (
	"chain/utils"
)

var (
	NodeMasterLogPath = utils.GetLogPath("NodeMaster")
	NodeAgentLogPath  = utils.GetLogPath("NodeAgent")

	NodeMasterAppDataPath = utils.GetAppDataPath("NodeMaster")
	NodeAgentAppDataPath  = utils.GetAppDataPath("NodeAgent")
)
