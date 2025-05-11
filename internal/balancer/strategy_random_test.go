package balancer_test

import (
	"testing"

	"github.com/mk/loadBalancer/internal/balancer"
)

func TestRandomStrategy_Next(t *testing.T) {
	// Инициализируем пул серверов с URL'ами
	pool := balancer.NewServerPool([]string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	})

	// Убедимся, что все бэкенды живые
	for _, backend := range pool.AllBackends() {
		backend.SetAlive(true)
	}

	// Назначаем стратегию Random
	pool.SetStrategy(balancer.NewRandomStrategy())

	// Проверим, что RandomStrategy возвращает живой бэкенд
	for i := 0; i < 10; i++ {
		selected := pool.NextBackend()
		if selected == nil {
			t.Fatal("Expected a backend, got nil")
		}
		if !selected.IsAlive() {
			t.Errorf("Selected backend is not alive: %+v", selected.URL)
		}
	}
}
