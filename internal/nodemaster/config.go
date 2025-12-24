package nodemaster

import (
	"chain/lib"
	"fmt"
)

type FirewallMode string

const (
	AllowOnly FirewallMode = "allow_only"
	AllowAll  FirewallMode = "allow_all"

	//local_only will override host part of addr with localhost
	LocalOnly FirewallMode = "local_only"
)

type Config struct {
	Addr    string
	AppName string

	FWMode         FirewallMode
	AllowOnlyHosts []string
}

var defaultConfig = Config{
	Addr:           ":8080",
	AppName:        "NodeMaster",
	FWMode:         AllowAll,
	AllowOnlyHosts: []string{},
}

func GetConfig() (Config, error) {
	cfg := defaultConfig
	var err error
	if cfg.FWMode == LocalOnly {
		cfg.Addr, err = lib.UpdateHost(cfg.Addr, "localhost")
		if err != nil {
			return Config{}, fmt.Errorf("failed to update addr '%s' of configuration for NodeMaster; err: %v", cfg.Addr, err.Error())
		}
	}
	return cfg, nil
}
