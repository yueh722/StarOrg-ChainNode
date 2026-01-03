package nodeproxy

type Config struct {
	AppName    string
	ServerAddr string
	HomeDir    string
	NodeID     string
}

var defaultConfig = Config{
	ServerAddr: "ws://localhost:8080/ws",
	AppName:    "NodeProxy",
	HomeDir:    "n:/cometbft",
}

func GetConfig() (Config, error) {
	return defaultConfig, nil
}
