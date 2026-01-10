package utils

import (
	"log"
	"os"
	"path/filepath"
)

// return %ProgramData%/appName or %appdata%/appName
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

func GetLogPath(appName string, moduleNames ...string) string {
	var basePath = os.Getenv("CHAIN_NODE_LOG_PATH")
	if basePath == "" {
		basePath = GetAppDataPath(appName)
	}

	if len(moduleNames) == 0 {
		basePath = filepath.Join(basePath, "log")
	} else {
		for _, m := range moduleNames {
			basePath = filepath.Join(basePath, m)
		}
		basePath = filepath.Join(basePath, "log")
	}

	logPath := basePath
	if logPath != "" {
		//TryCreateDir(logPath)
	}

	return logPath
}

func TryCreateDir(path string) string {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatalf("create directory %s, error: %v", path, err)
	}
	return path
}
