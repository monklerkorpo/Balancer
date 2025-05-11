
package balancer

import (
	"testing"
)

func TestLeastConnectionsStrategy(t *testing.T) {
	pool := NewServerPool([]string{"http://a", "http://b", "http://c"})
	for _, b := range pool.AllBackends() {
		b.SetAlive(true)
	}

	// Установим разные количества соединений
	pool.AllBackends()[0].ActiveConnections = 5 // a
	pool.AllBackends()[1].ActiveConnections = 2 // b
	pool.AllBackends()[2].ActiveConnections = 7 // c

	pool.SetStrategy(NewLeastConnectionsStrategy())

	backend := pool.NextBackend()
	if backend == nil {
		t.Fatal("expected backend, got nil")
	}

	if backend.URL != "http://b" {
		t.Errorf("expected http://b with least connections, got %s", backend.URL)
	}
}
