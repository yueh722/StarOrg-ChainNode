package common

const (
	KeyClientID = "ClientID"
)

type ClientContext struct {
	ClientID string
	Token    string
	SeqNum   uint64
}

type KeepUntilDead struct {
	ServerID string
	ClientID string
	SeqNum   uint64
}
