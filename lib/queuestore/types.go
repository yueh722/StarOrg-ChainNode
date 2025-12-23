package queuestore

type PersistedEvent struct {
	EventID string `json:"event_id"` // boot_id:seq
	NodeID  string `json:"node_id"`
	Seq     uint64 `json:"seq"`
	Level   string `json:"level"`
	Time    int64  `json:"time"` // UnixMilli
	JSON    string `json:"json"` // zap json string (TEXT)
}

type QueueStore interface {
	Put(e PersistedEvent) error
	MarkSent(eventID string) error
	ScanUnsent(limit int, fn func(PersistedEvent) error) error
	GetBroker() *Broker
	Close() error
}
