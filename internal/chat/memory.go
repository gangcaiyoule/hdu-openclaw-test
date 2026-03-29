package chat

import "sync"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MemoryStore struct {
	mu      sync.RWMutex
	limit   int
	history map[string][]Message
}

func NewMemoryStore(limit int) *MemoryStore {
	return &MemoryStore{
		limit:   limit,
		history: make(map[string][]Message),
	}
}

func (m *MemoryStore) Get(sessionID string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := m.history[sessionID]
	out := make([]Message, len(items))
	copy(out, items)
	return out
}

func (m *MemoryStore) Append(sessionID string, messages ...Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := append(m.history[sessionID], messages...)
	if m.limit > 0 && len(items) > m.limit*2 {
		items = items[len(items)-m.limit*2:]
	}
	m.history[sessionID] = items
}
