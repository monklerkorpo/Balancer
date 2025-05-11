package balancer

import (
	"testing"
)

func TestRoundRobinStrategy(t *testing.T) {
	pool := NewServerPool([]string{"http://a", "http://b", "http://c"})
	for _, b := range pool.AllBackends() {
		b.SetAlive(true)
	}

	pool.SetStrategy(NewRoundRobinStrategy())

	got := []string{}
	for i := 0; i < 6; i++ {
		backend := pool.NextBackend()
		if backend == nil {
			t.Fatalf("expected backend, got nil")
		}
		got = append(got, backend.URL)
	}

	expected := []string{"http://a", "http://b", "http://c", "http://a", "http://b", "http://c"}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("round robin mismatch at %d: got %s, want %s", i, got[i], expected[i])
		}
	}
}
