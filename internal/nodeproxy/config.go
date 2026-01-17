package nodeproxy

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/utils"
	"fmt"
	"path/filepath"

	"github.com/duke-git/lancet/v2/fileutil"
	_ "github.com/spf13/viper/remote"
	"gopkg.in/ini.v1"
)

type Config struct {
	AppName    string
	ServerAddr string
	HomeDir    string
	BinDir     string // if leave blank, HomeDir/Bin will be applied.
}

var defaultConfig = Config{
	ServerAddr: "ws://localhost:8080/ws",
	AppName:    "NodeProxy",
	HomeDir:    filepath.Join(utils.GetAppDataPath("node")),
}

func GetDefaultConfig() Config {
	if defaultConfig.BinDir == "" {
		defaultConfig.BinDir = filepath.Join(defaultConfig.HomeDir, "bin")
	}
	return defaultConfig
}

func LoadConfig(cfgFiles []string, defaultCfg Config) (*Config, error) {

	if len(cfgFiles) == 0 {
		zlog.Info("Default config applied")
		return &defaultCfg, nil
	}

	for _, cfgFile := range cfgFiles {
		if !fileutil.IsExist(cfgFile) {
			continue
		}

		iniFile, err := ini.Load(cfgFile)
		if err != nil {
			return nil, fmt.Errorf("read config file error: %v", err)
		}

		defaultSection := iniFile.Section("default")

		cfg := &Config{}
		if err := defaultSection.MapTo(cfg); err != nil {
			return nil, fmt.Errorf("data in config file error: ", err)
		}

		zlog.Infof("config file %s applied", cfgFile)

		if cfg.BinDir == "" {
			cfg.BinDir = filepath.Join(cfg.HomeDir, "bin")
		}

		if cfg.AppName == "" {
			newAppName := lib.RandomNodeName()
			defaultSection.Key("AppName").SetValue(newAppName)
			if err := iniFile.SaveTo(cfgFile); err != nil {
				return nil, fmt.Errorf("save config file of NodeProxy error: %v", err)
			}

			zlog.Info("create a random node name with current config", newAppName)
		}

		return cfg, nil
	}

	return &defaultCfg, nil
}
