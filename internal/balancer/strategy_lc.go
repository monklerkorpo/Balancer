package balancer


type LeastConnectionsStrategy struct{}

func NewLeastConnectionsStrategy() Strategy {
    return &LeastConnectionsStrategy{}
}


// Next выбирает живой бэкенд с наименьшим числом активных соединений.
func (s *LeastConnectionsStrategy) Next(p *ServerPool) *Backend {
	alive := p.GetAliveBackends()
	if len(alive) == 0 {
		return nil
	}

	var min *Backend
	minConnections := int(^uint(0) >> 1) // максимальное значение int

	for _, b := range alive {
		conns := b.GetConnections()
		if min == nil || conns < minConnections {
			min = b
			minConnections = conns
		}
	}

	return min
}


