package nodemaster

import (
	"chain/lib"
	"chain/rpc/security"
	"chain/rpc/server"
	"fmt"
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
	AllowOnlyHosts: []string{},
	Tls:            testTlsOption,
}

func GetConfig() (Config, error) {
	cfg := allowOnlyConfig
	var err error
	if cfg.FWMode == security.LocalOnly {
		cfg.Addr, err = lib.UpdateHost(cfg.Addr, "localhost")
		if err != nil {
			return Config{}, fmt.Errorf("failed to update addr '%s' of configuration for NodeMaster; err: %v", cfg.Addr, err.Error())
		}
	}
	return cfg, nil
}
