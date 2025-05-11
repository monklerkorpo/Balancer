package balancer

import (
	"sync"
)

// Backend представляет сервер с флагом доступности и количеством активных соединений.
type Backend struct {
	URL               string
	Alive             bool
	ActiveConnections int
	mu                sync.RWMutex
}

// NewBackend создает новый экземпляр Backend.
func NewBackend(url string) *Backend {
	return &Backend{
		URL:   url,
		Alive: true,
	}
}

// SetAlive обновляет статус доступности.
func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Alive = alive
}

// IsAlive возвращает текущий статус доступности.
func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

// IncConnections увеличивает количество активных соединений.
func (b *Backend) IncConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ActiveConnections++
}

// DecConnections уменьшает количество активных соединений.
func (b *Backend) DecConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ActiveConnections > 0 {
		b.ActiveConnections--
	}
}

// GetConnections возвращает текущее количество активных соединений.
func (b *Backend) GetConnections() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ActiveConnections
}

// ResetConnections сбрасывает счетчик активных соединений.
func (b *Backend) ResetConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ActiveConnections = 0
}
