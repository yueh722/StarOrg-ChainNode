package queuestore

import "sync"

type Broker struct {
	mu   sync.RWMutex
	subs map[chan PersistedEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[chan PersistedEvent]struct{})}
}

func (b *Broker) Subscribe(buf int) chan PersistedEvent {
	ch := make(chan PersistedEvent, buf)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(ch chan PersistedEvent) {
	b.mu.Lock()
	if _, ok := b.subs[ch]; ok {
		delete(b.subs, ch)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *Broker) Publish(e PersistedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
			// 慢订阅者：丢（你也可以改为踢掉）
		}
	}
}
