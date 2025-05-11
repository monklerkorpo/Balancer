package ratelimiter

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/mk/loadBalancer/internal/ratelimiter"
	"github.com/mk/loadBalancer/internal/storage"
	"go.uber.org/zap"
)

func setupTestRateLimiter(capacity, rate int) *ratelimiter.RateLimiter {
	logger := zap.NewNop().Sugar()

	repo, err := storage.NewSQLiteClientRepo("file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("не удалось создать SQLiteClientRepo: %v", err)
	}

	return ratelimiter.NewRateLimiter(capacity, rate, repo, logger)
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := setupTestRateLimiter(1000, 100)
	clientID := "bench_client"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.AllowRequest(clientID)
		}
	})
}

func BenchmarkRateLimiterWithMultipleClients(b *testing.B) {
	rl := setupTestRateLimiter(100, 10)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			clientID := fmt.Sprintf("client_%d", i%10)
			rl.AllowRequest(clientID)
			i++
		}
	})
}

func TestRateLimiter_BasicLimit(t *testing.T) {
	rl := setupTestRateLimiter(5, 1)
	clientID := "client1"

	for i := 0; i < 5; i++ {
		if !rl.AllowRequest(clientID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	if rl.AllowRequest(clientID) {
		t.Error("Expected request to be blocked")
	}

	time.Sleep(1200 * time.Millisecond)

	if !rl.AllowRequest(clientID) {
		t.Error("Request after refill should be allowed")
	}
}

func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := setupTestRateLimiter(3, 1)

	for i := 0; i < 3; i++ {
		if !rl.AllowRequest("client1") {
			t.Errorf("Client1 request %d should be allowed", i+1)
		}
	}

	if !rl.AllowRequest("client2") {
		t.Error("Client2 first request should be allowed")
	}
}

