package balancer

import (
	"math/rand"
	"time"
)

type RandomStrategy struct{}

func NewRandomStrategy() Strategy {
	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())
	return &RandomStrategy{}
}

// Next выбирает случайный живой бэкенд
func (s *RandomStrategy) Next(p *ServerPool) *Backend {
	alive := p.GetAliveBackends()
	if len(alive) == 0 {
		return nil
	}
	return alive[rand.Intn(len(alive))]
}
