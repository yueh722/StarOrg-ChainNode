package nodeproxy

type Config struct {
	AppName    string
	ServerAddr string
	NodeDir    string
}

var defaultConfig = Config{
	ServerAddr: "wss://localhost:8080/ws",
	AppName:    "NodeProxy",
	NodeDir:    "n:/cometbft",
}

func GetConfig() (Config, error) {
	return defaultConfig, nil
}
