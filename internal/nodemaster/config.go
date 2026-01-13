package nodemaster

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/security"
	"chain/rpc/server"
	"fmt"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/spf13/viper"
)

type Config struct {
	Addr    string
	AppName string
	TlsPort string

	FWMode         security.FirewallMode
	AllowOnlyHosts []string

	Tls server.TlsOption
}

var testTlsOption = server.TlsOption{
	TlsCertFile: "n:/tmp/test.crt",
	TlsKeyFile:  "n:/tmp/test.key",
}

var defaultConfig = Config{
	Addr:           ":8080",
	AppName:        "NodeMaster",
	FWMode:         security.AllowAll,
	AllowOnlyHosts: []string{},
	Tls:            testTlsOption,
}

var localOnlyConfig = Config{
	Addr:           ":8080",
	AppName:        "NodeMaster",
	FWMode:         security.LocalOnly,
	AllowOnlyHosts: []string{},
	Tls:            testTlsOption,
}

var allowOnlyConfig = Config{
	Addr:           ":8080",
	AppName:        "NodeMaster",
	FWMode:         security.AllowOnly,
	AllowOnlyHosts: []string{"127.0.0.1", "192.168.1.5"},
	Tls:            testTlsOption,
}

func GetDefaultConfig() Config {
	cfg := allowOnlyConfig
	if cfg.FWMode == security.LocalOnly {
		cfg.Addr, _ = lib.UpdateHost(cfg.Addr, "localhost")
	}
	return cfg
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

		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config file error: ", err)
		}

		cfg := &Config{}
		if err := viper.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("data in config file error: ", err)
		}

		zlog.Infof("config file %s applied", cfgFile)
		return cfg, nil
	}
	zlog.Info("Default config applied")
	return &defaultCfg, nil
}
