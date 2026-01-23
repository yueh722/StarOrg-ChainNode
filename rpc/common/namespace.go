package common

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
)

const (
	NameSpaceSetupService = "chain"
)

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

type LiveTxNamespace struct{}

func (s LiveTxNamespace) GetNamesapceString() string {
	return "liveTx"
}
func (s LiveTxNamespace) GetSubscriptionName() string {
	return ""
}
