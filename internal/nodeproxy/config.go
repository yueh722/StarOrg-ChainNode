package nodeproxy

import (
	"chain/lib/zlog"
	"fmt"

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
	HomeDir:    "n:/cometbft",
}

func GetDefaultConfig() Config {
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

		cfg := &Config{}
		if err := iniFile.Section("default").MapTo(cfg); err != nil {
			return nil, fmt.Errorf("data in config file error: ", err)
		}

		zlog.Infof("config file %s applied", cfgFile)
		return cfg, nil
	}

	return &defaultCfg, nil
}
