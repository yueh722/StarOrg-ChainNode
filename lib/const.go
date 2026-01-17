package lib

const (
	AppFilePath  = "bin/abci-v2-Identity-app.exe"
	NodeFilePath = "bin/cometbft.exe"

	AppFileName  = "abci-v2-Identity-app.exe"
	NodeFileName = "cometbft.exe"
)

const DelegatorApp = "NodeProxyD_"

var (
	DefaultP2PPort        uint16 = 26656
	DefaultRPCPort        uint16 = 26657
	DefaultABCIPort       uint16 = 26658
	DefaultWebServicePort uint16 = 1000
)

func CorrectP2PPortIfNeedTo(port *uint16) {
	if *port == 0 {
		*port = DefaultP2PPort
	}
}

func CorrectRPCPortIfNeedTo(port *uint16) {
	if *port == 0 {
		*port = DefaultRPCPort
	}
}

func CorrectABCIPortIfNeedTo(port *uint16) {
	if *port == 0 {
		*port = DefaultABCIPort
	}
}

func CorrectWebServicePortIfNeedTo(port *uint16) {
	if *port == 0 {
		*port = DefaultWebServicePort
	}
}
