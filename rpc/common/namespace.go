package common

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
)

const (
	NameSpaceSetupService = "chain"
)

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
