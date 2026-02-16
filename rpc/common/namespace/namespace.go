package namespace

import (
	"fmt"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
)

var rpcMethods = map[string]string{}

func init() {

	methodsMap := map[string][]string{
		//client
		//StatusClient
		StatusClientNamespace{}.GetNamesapceString(): []string{"IsNodeRpcAlive"},

		/////////////////////////////////////////////
		//server
		//setup service
		SetupNamespace{}.GetNamesapceString(): []string{
			"GetChainState",
			"BroadcastInitializeChainNode",
			"BroadcastAddChainNode",
			"ControlNode",
			"BroadcastPurgeChainNode",
			"ReportCommandResult",
			"InitializationComplete",
			"GetHistoryCommands",
			"GetNodeInitializedStatus",
		},

		//status service
		StatusNamespace{}.GetNamesapceString(): []string{
			"ReportNodeStatus",
			"ReportStorageStatus",
			"ReportCometbftProcessStatus",
		},

		//livetx service
		LiveTxNamespace{}.GetNamesapceString(): []string{
			"EnableTransmitter",
			"ReportLiveTx",
		},

		//stream service
		StreamNamespace{}.GetNamesapceString(): []string{
			"ReportState",
		},

		//nodelog service
		NodeLogNamespace{}.GetNamesapceString(): []string{
			"ReportLog",
		},
	}

	for namespace, methods := range methodsMap {
		slice.ForEach(methods, func(index int, name string) {
			rpcMethods[name] = MakeMethodName(namespace, name)
		})
	}
}

type Namespace interface {
	GetNamesapceString() string
	GetSubscriptionName() string
}

type StreamNamespace struct{}

func (s StreamNamespace) GetNamesapceString() string {
	return "stream"
}
func (s StreamNamespace) GetSubscriptionName() string {
	return "subscribeStream"
}

type SetupNamespace struct{}

func (s SetupNamespace) GetNamesapceString() string {
	return "chain"
}

func (s SetupNamespace) GetSubscriptionName() string {
	return "subscribeCommand"
}

type NodeLogNamespace struct{}

func (s NodeLogNamespace) GetNamesapceString() string {
	return "nodelog"
}

func (s NodeLogNamespace) GetSubscriptionName() string {
	return ""
}

func MakeMethodName(nameSpace, method string) string {
	if nameSpace == "" || method == "" {
		return ""
	}

	return fmt.Sprintf("%s_%s", nameSpace, strutil.LowerFirst(method))
}

func GetMethodName(method string) string {
	return rpcMethods[method]
}

type LiveTxNamespace struct{}

func (s LiveTxNamespace) GetNamesapceString() string {
	return "liveTx"
}
func (s LiveTxNamespace) GetSubscriptionName() string {
	return ""
}

type StatusNamespace struct{}

func (s StatusNamespace) GetNamesapceString() string {
	return "status"
}
func (s StatusNamespace) GetSubscriptionName() string {
	return ""
}

type TickerNamespace struct{}

func (s TickerNamespace) GetNamesapceString() string {
	return "ticker"
}
func (s TickerNamespace) GetSubscriptionName() string {
	return ""
}

type StatusClientNamespace struct{}

func (s StatusClientNamespace) GetNamesapceString() string {
	return "statusclient"
}
func (s StatusClientNamespace) GetSubscriptionName() string {
	return ""
}
