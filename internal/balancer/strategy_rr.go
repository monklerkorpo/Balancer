package balancer

import (
	"sync/atomic"

)

type RoundRobin struct {}

func NewRoundRobinStrategy() Strategy {
    return &RoundRobin{}
}

// Next выбирает следующий живой бэкенд по Round-Robin.
func (r *RoundRobin) Next(p *ServerPool) *Backend {
	alive := p.GetAliveBackends()
	n := len(alive)
	if n == 0 {
		return nil
	}
	idx := int(atomic.LoadUint64(&p.current) % uint64(n))
	atomic.AddUint64(&p.current, 1)
	return alive[idx]
}
