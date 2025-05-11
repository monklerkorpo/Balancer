package ratelimiter

import (
	"sync"
	"time"

	"github.com/mk/loadBalancer/internal/storage"
	"go.uber.org/zap"
)

type RateLimiter struct {
	mu            sync.Mutex
	buckets       map[string]*tokenBucket
	repo          storage.ClientRepository
	defaultCap    int
	defaultRefill int
	logger        *zap.SugaredLogger
}

type ClientLimit struct {
	Capacity   int
	RefillRate int
}

func NewRateLimiter(capacity, refillRate int, repo storage.ClientRepository, logger *zap.SugaredLogger) *RateLimiter {
	rl := &RateLimiter{
		buckets:       make(map[string]*tokenBucket),
		repo:          repo,
		defaultCap:    capacity,
		defaultRefill: refillRate,
		logger:        logger,
	}
	go rl.refillTokens()
	return rl
}

func (rl *RateLimiter) AllowRequest(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[clientID]
	if !ok {
		bucket = rl.initBucketForClient(clientID)
	}
	return bucket.consume()
}

func (rl *RateLimiter) initBucketForClient(clientID string) *tokenBucket {
	limit, err := rl.repo.Get(clientID)
	capacity := rl.defaultCap
	refill := rl.defaultRefill

	if err == nil {
		capacity = limit.Capacity
		refill = limit.RefillRate
	} else {
		rl.logger.Warnw("Failed to fetch rate limit from repository, using default values",
			"client_id", clientID, "error", err)
	}

	bucket := newTokenBucket(capacity, refill)
	rl.buckets[clientID] = bucket
	return bucket
}

func (rl *RateLimiter) refillTokens() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for _, bucket := range rl.buckets {
			bucket.refill()
		}
		rl.mu.Unlock()
	}
}

type tokenBucket struct {
	capacity     int
	refillRate   int
	tokens       int
	lastRefilled time.Time
}

func newTokenBucket(capacity, refillRate int) *tokenBucket {
	return &tokenBucket{
		capacity:     capacity,
		refillRate:   refillRate,
		tokens:       capacity,
		lastRefilled: time.Now(),
	}
}

func (b *tokenBucket) consume() bool {
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

func (b *tokenBucket) refill() {
	now := time.Now()
	elapsed := int(now.Sub(b.lastRefilled).Seconds())
	if elapsed <= 0 {
		return
	}

	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefilled = now
}

func (rl *RateLimiter) SetClientLimit(clientID string, limit ClientLimit) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[clientID]
	if ok {
		bucket.capacity = limit.Capacity
		bucket.refillRate = limit.RefillRate
	} else {
		rl.buckets[clientID] = newTokenBucket(limit.Capacity, limit.RefillRate)
	}
}

func (rl *RateLimiter) RemoveClient(clientID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, clientID)
}
