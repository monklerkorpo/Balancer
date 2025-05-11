package balancer

import (
	"context"
	"net/http"
	"time"

)

// Checker выполняет периодические health checks для бэкендов.
type Checker struct {
	Backends []*Backend
	Interval time.Duration
	Client   *http.Client
}

// NewChecker создаёт новый Checker с заданным списком бэкендов и интервалом.
func NewChecker(backends []*Backend, interval time.Duration) *Checker {
	return &Checker{
		Backends: backends,
		Interval: interval,
		Client:   &http.Client{Timeout: 2 * time.Second},
	}
}

// Run запускает цикл проверок доступности бэкендов до отмены контекста.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, b := range c.Backends {
				go c.checkBackend(b)
			}
		case <-ctx.Done():
			return
		}
	}
}

// checkBackend отправляет GET-запрос на /healthz и обновляет статус Alive у бэкенда.
func (c *Checker) checkBackend(b *Backend) {
	resp, err := c.Client.Get(b.URL + "/healthz")
	alive := err == nil && resp.StatusCode == http.StatusOK
	b.SetAlive(alive)
	if resp != nil {
		resp.Body.Close()
	}
}
