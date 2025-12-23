package queuestore

import (
	"errors"
	"sort"
	"sync"
)

var ErrEmptyEventID = errors.New("event_id empty")

type InMemoryStore struct {
	mu     sync.Mutex
	m      map[string]*memItem
	broker *Broker
}

type memItem struct {
	e    PersistedEvent
	sent bool
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		m:      make(map[string]*memItem),
		broker: NewBroker(),
	}
}

func (s *InMemoryStore) GetBroker() *Broker {
	return s.broker
}

func (s *InMemoryStore) Put(e PersistedEvent) error {
	if e.EventID == "" {
		return ErrEmptyEventID
	}

	s.broker.Publish(e)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.m[e.EventID]; ok {
		return nil // idempotent
	}
	cp := e
	s.m[e.EventID] = &memItem{e: cp, sent: false}
	return nil
}

func (s *InMemoryStore) MarkSent(eventID string) error {
	if eventID == "" {
		return ErrEmptyEventID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if it, ok := s.m[eventID]; ok {
		it.sent = true
	}
	return nil // idempotent
}

func (s *InMemoryStore) ScanUnsent(limit int, fn func(PersistedEvent) error) error {
	s.mu.Lock()
	list := make([]PersistedEvent, 0, len(s.m))
	for _, it := range s.m {
		if !it.sent {
			list = append(list, it.e)
		}
	}
	s.mu.Unlock()

	sort.Slice(list, func(i, j int) bool {
		if list[i].Time != list[j].Time {
			return list[i].Time < list[j].Time
		}
		return list[i].Seq < list[j].Seq
	})

	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	for _, e := range list {
		if err := fn(e); err != nil {
			return err
		}
	}
	return nil
}

func (s *InMemoryStore) Close() error { return nil }
