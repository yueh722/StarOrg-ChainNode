package nodeproxy

type Config struct {
	AppName    string
	ServerAddr string
}

var defaultConfig = Config{
	ServerAddr: "ws://localhost:8080/ws",
	AppName:    "NodeProxy",
}

func GetConfig() (Config, error) {
	return defaultConfig, nil
}
