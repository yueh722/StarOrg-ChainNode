package common

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
