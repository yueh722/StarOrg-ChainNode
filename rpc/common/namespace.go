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

func MakeMethodName(nameSpace, method string) string {
	if nameSpace == "" || method == "" {
		return ""
	}

	return fmt.Sprintf("%s_%s", nameSpace, strutil.LowerFirst(method))
}
