// Package storage keeps incidents. The MVP uses an in-memory, concurrency-safe store
// behind a small interface so a SQLite or Postgres implementation can drop in later
// without touching the rest of the system - persistence is a stretch goal, not a
// prerequisite for the deterministic core.
package storage

import (
	"sort"
	"sync"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// Store is the persistence boundary for incidents.
type Store interface {
	Put(inc *models.Incident)
	Get(id string) (*models.Incident, bool)
	List() []*models.Incident
}

// Memory is an in-memory Store, safe for concurrent use.
type Memory struct {
	mu   sync.RWMutex
	data map[string]*models.Incident
}

// NewMemory builds an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{data: map[string]*models.Incident{}}
}

// Put inserts or replaces an incident.
func (m *Memory) Put(inc *models.Incident) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[inc.ID] = inc
}

// Get returns an incident by ID.
func (m *Memory) Get(id string) (*models.Incident, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inc, ok := m.data[id]
	return inc, ok
}

// List returns all incidents, most recently created first.
func (m *Memory) List() []*models.Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*models.Incident, 0, len(m.data))
	for _, inc := range m.data {
		out = append(out, inc)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
