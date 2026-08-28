package app

import (
	"sync"

	"github.com/hellojinjie/TunnelDock/Windows/internal/model"
)

type Model struct {
	mu          sync.RWMutex
	hosts       []model.SSHHost
	searchQuery string
}

func NewModel() *Model { return &Model{} }

func (m *Model) SetHosts(hosts []model.SSHHost) {
	m.mu.Lock()
	m.hosts = append([]model.SSHHost(nil), hosts...)
	m.mu.Unlock()
}

func (m *Model) SetSearchQuery(query string) {
	m.mu.Lock()
	m.searchQuery = query
	m.mu.Unlock()
}

func (m *Model) FilteredHosts() []model.SSHHost {
	m.mu.RLock()
	hosts := append([]model.SSHHost(nil), m.hosts...)
	query := m.searchQuery
	m.mu.RUnlock()
	return FilterHosts(hosts, query)
}

// Hosts returns the unfiltered SSH host snapshot for secondary presentation
// such as the missing-host section in the native sidebar.
func (m *Model) Hosts() []model.SSHHost {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.SSHHost(nil), m.hosts...)
}

func (m *Model) SearchQuery() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.searchQuery
}
