package chat

import "sync"

// Message 表示一轮短时会话上下文中的消息。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MemoryStore 按会话保存有上限的内存上下文历史。
type MemoryStore struct {
	mu      sync.RWMutex
	limit   int
	history map[string][]Message
}

// NewMemoryStore 创建一个带会话上限的内存历史存储。
func NewMemoryStore(limit int) *MemoryStore {
	return &MemoryStore{
		limit:   limit,
		history: make(map[string][]Message),
	}
}

// Get 返回指定会话历史的副本。
func (m *MemoryStore) Get(sessionID string) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := m.history[sessionID]
	out := make([]Message, len(items))
	copy(out, items)
	return out
}

// Append 向会话中追加新消息，并在超出上限时裁剪旧历史。
func (m *MemoryStore) Append(sessionID string, messages ...Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := append(m.history[sessionID], messages...)
	if m.limit > 0 && len(items) > m.limit*2 {
		items = items[len(items)-m.limit*2:]
	}
	m.history[sessionID] = items
}
