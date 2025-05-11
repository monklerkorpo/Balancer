package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mk/loadBalancer/internal/balancer"
)

func TestHealthCheckerIntegration(t *testing.T) {
	// Создаём живой сервер
	liveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer liveServer.Close()

	// Создаём мертвый сервер (ничего не запускаем на этом порту)
	deadServerURL := "http://127.0.0.1:65534"

	// Создаем бэкенды
	backends := []*balancer.Backend{
		balancer.NewBackend(liveServer.URL),
		balancer.NewBackend(deadServerURL),
	}

	// Создаем checker с малым интервалом
	checker := balancer.NewChecker(backends, 100*time.Millisecond)

	// Контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Запускаем проверки
	checker.Run(ctx)

	// Ждём чуть больше чем интервал (чтобы успел хотя бы один цикл пройти)
	time.Sleep(600 * time.Millisecond)

	// Проверяем статус бэкендов
	if !backends[0].IsAlive() {
		t.Errorf("Expected liveServer to be alive")
	}
	if backends[1].IsAlive() {
		t.Errorf("Expected deadServer to be dead")
	}
}
