package session

import (
	"slices"
	"sync"
	"time"
)

// MemoryStorage provides an in-memory implementation of Storage for testing.
// All data is stored in memory and lost when the instance is destroyed.
type MemoryStorage struct {
	mu       sync.RWMutex
	sessions map[string]*memorySession
}

// memorySession holds a session's data in memory.
type memorySession struct {
	sessionType string // "raw" or "events"
	meta        *StoreMeta
	entries     []map[string]any
	createdAt   time.Time
	modTime     time.Time
	size        int64
}

// NewMemoryStorage creates a new in-memory storage for testing.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		sessions: make(map[string]*memorySession),
	}
}

// Ensure MemoryStorage implements Storage interface.
var _ Storage = (*MemoryStorage)(nil)

// Save implements Storage.Save for in-memory storage.
func (m *MemoryStorage) Save(filename, sessionType string, meta *StoreMeta, entries []map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// estimate size (rough approximation)
	var size int64
	for _, e := range entries {
		size += int64(len(e) * 50) // rough estimate per entry
	}

	// use filename as key
	t := &memorySession{
		sessionType: sessionType,
		meta:        meta,
		entries:     entries,
		createdAt:   now,
		modTime:     now,
		size:        size,
	}
	m.sessions[filename] = t

	return nil
}

// Load implements Storage.Load for in-memory storage.
func (m *MemoryStorage) Load(filename string) (*StoredSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, ok := m.sessions[filename]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return &StoredSession{
		Info: SessionInfo{
			Filename:  filename,
			FilePath:  "memory://" + filename,
			Type:      data.sessionType,
			Size:      data.size,
			CreatedAt: data.createdAt,
			ModTime:   data.modTime,
		},
		Meta:    data.meta,
		Entries: data.entries,
	}, nil
}

// List implements Storage.List for in-memory storage.
func (m *MemoryStorage) List() ([]SessionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []SessionInfo
	for filename, data := range m.sessions {
		infos = append(infos, SessionInfo{
			Filename:  filename,
			FilePath:  "memory://" + filename,
			Type:      data.sessionType,
			Size:      data.size,
			CreatedAt: data.createdAt,
			ModTime:   data.modTime,
		})
	}

	// sort by mod time descending (newest first)
	slices.SortFunc(infos, func(a, b SessionInfo) int {
		return b.ModTime.Compare(a.ModTime)
	})

	return infos, nil
}

// Delete implements Storage.Delete for in-memory storage.
func (m *MemoryStorage) Delete(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[filename]; !ok {
		return ErrSessionNotFound
	}

	delete(m.sessions, filename)
	return nil
}

// Exists implements Storage.Exists for in-memory storage.
func (m *MemoryStorage) Exists(filename string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.sessions[filename]
	return ok
}

// GetLatest implements Storage.GetLatest for in-memory storage.
func (m *MemoryStorage) GetLatest() (*SessionInfo, error) {
	infos, err := m.List()
	if err != nil {
		return nil, err
	}

	if len(infos) == 0 {
		return nil, ErrNoSessions
	}

	return &infos[0], nil
}

// Clear removes all sessions from memory (useful for test cleanup).
func (m *MemoryStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*memorySession)
}

// Count returns the number of sessions in memory.
func (m *MemoryStorage) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}
