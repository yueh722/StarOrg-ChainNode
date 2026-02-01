package nodemaster

import (
	"chain/lib"
	"chain/lib/zlog"
	"chain/rpc/security"
	"chain/rpc/server"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/fileutil"
	"gopkg.in/ini.v1"
)

type Config struct {
	Addr    string
	AppName string
	//TlsPort string

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

func (cfg *Config) Valid() error {

	//check addr
	parts := strings.SplitN(cfg.Addr, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid Server address: %s", cfg.Addr)
	}

	if cfg.AppName == "" {
		cfg.AppName = "NodeMaster"
	}

	if cfg.FWMode == security.LocalOnly {
		cfg.Addr, _ = lib.UpdateHost(cfg.Addr, "localhost")
		cfg.AllowOnlyHosts = append(cfg.AllowOnlyHosts, "127.0.0.1", "localhost")

		//distinct slice 'AllowOnlyHosts' if u think need to
	}

	return nil
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

		iniFile, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, cfgFile)
		if err != nil {
			return nil, fmt.Errorf("read config file error: %v", err)
		}

		cfg := &Config{}
		if err := iniFile.Section("default").MapTo(cfg); err != nil {
			return nil, fmt.Errorf("failed to map default section in ini file %v, error: %v", cfgFile, err)
		}

		if err := iniFile.Section("firewall").MapTo(cfg); err != nil {
			return nil, fmt.Errorf("failed to map  firewall section in ini file %v, error: %v", cfgFile, err)
		}

		if err := iniFile.Section("tls").MapTo(&cfg.Tls); err != nil {
			return nil, fmt.Errorf("failed to map tls section in ini file %v, error: %v", cfgFile, err)
		}

		if err := cfg.Valid(); err != nil {
			return nil, fmt.Errorf("config data is invalidate, err: %v", err)
		}

		zlog.Infof("config file %s applied", cfgFile)
		return cfg, nil
	}
	zlog.Info("Default config applied")
	return &defaultCfg, nil
}
