package notification

import "sync"

const (
	MsgTypeOffline       = "offline"
	MsgTypeOnline        = "online"
	MsgTypeInitialized   = "initialized"
	MsgTypeUnInitialized = "uninitialized"
)

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Time    int64       `json:"time"` // Unix timestamp
}

type Listener interface {
	OnNotify(msg *Message)
}

type Manager struct {
	mu        sync.RWMutex
	listeners []Listener
}

func NewManager(registers ...Listener) *Manager {
	mgr := &Manager{
		listeners: make([]Listener, 0),
	}

	for _, register := range registers {
		mgr.Register(register)
	}

	return mgr
}

func (m *Manager) Register(listener Listener) {
	if listener == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

func (m *Manager) Unregister(listener Listener) {
	if listener == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, l := range m.listeners {
		if l == listener {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			break
		}
	}
}

func (m *Manager) Notify(msg Message) {
	m.mu.RLock()
	listeners := make([]Listener, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.RUnlock()

	for _, listener := range listeners {
		// go listener.Notify(msg)
		listener.OnNotify(&msg)
	}
}
