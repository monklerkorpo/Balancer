package balancer

import (
	"sync"
)

// ServerPool управляет всеми бэкендами и стратегией выбора.
type ServerPool struct {
	backends []*Backend
	current  uint64       // для round-robin
	strategy Strategy
	mu       sync.RWMutex
}

// NewServerPool создаёт новый пул серверов с заданными URL.
func NewServerPool(urls []string) *ServerPool {
	backends := make([]*Backend, 0, len(urls))
	for _, url := range urls {
		backends = append(backends, NewBackend(url))
	}
	return &ServerPool{
		backends: backends,
	}
}

// SetStrategy задаёт стратегию выбора бэкенда.
func (p *ServerPool) SetStrategy(s Strategy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategy = s
}

// GetStrategy возвращает текущую стратегию.
func (p *ServerPool) GetStrategy() Strategy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.strategy
}

// NextBackend возвращает следующий бэкенд согласно стратегии.
func (p *ServerPool) NextBackend() *Backend {
	p.mu.RLock()
	strategy := p.strategy
	p.mu.RUnlock()

	if strategy == nil {
		return nil
	}
	return strategy.Next(p)
}

// GetAliveBackends возвращает список живых бэкендов.
func (p *ServerPool) GetAliveBackends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()

	alive := make([]*Backend, 0)
	for _, b := range p.backends {
		if b.IsAlive() {
			alive = append(alive, b)
		}
	}
	return alive
}

// AllBackends возвращает все бэкенды (живые и мертвые).
func (p *ServerPool) AllBackends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.backends
}

// AddBackend добавляет новый бэкенд в пул.
func (p *ServerPool) AddBackend(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.backends = append(p.backends, NewBackend(url))
}

// MarkBackendAlive обновляет статус живости бэкенда по URL.
func (p *ServerPool) MarkBackendAlive(url string, alive bool) {
	p.mu.Lock() // используем Lock, так как мы меняем состояние
	defer p.mu.Unlock() // необходимо использовать Unlock

	for _, b := range p.backends {
		if b.URL == url {
			b.SetAlive(alive)
			break
		}
	}
}


// ResetConnections сбрасывает количество соединений у всех бэкендов.
func (p *ServerPool) ResetConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, b := range p.backends {
		b.ResetConnections()
	}
}
