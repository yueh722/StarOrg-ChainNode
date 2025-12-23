package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetAppDataPath(appName string) string {
	appData := os.Getenv("ProgramData")
	if appData == "" {
		appData, _ = os.UserConfigDir()
	}

	return filepath.Join(appData, "chainnode", appName)
}

func GetStoragePath() string {
	var basePath = os.Getenv("CHAIN_NODE_STORAGE_PATH")
	if basePath == "" {
		basePath = filepath.Join(GetAppDataPath(""), "storage")
	}

	return basePath
}

func GetStorageDrive() string {
	basePath := GetStoragePath()
	return basePath[:len("c:/")]
}

func GetLogPath(appName, moduleName string) string {
	var basePath = os.Getenv("CHAIN_NODE_LOG_PATH")
	if basePath == "" {
		basePath = GetAppDataPath(appName)
	}

	basePath = filepath.Join(basePath, "log")

	logPath := filepath.Join(basePath, moduleName+".log")
	if logPath != "" {
		if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
			fmt.Printf("can not create log directory: %v, err: %v\n", logPath, err)
			os.Exit(100)
		}
	}

	return logPath
}
