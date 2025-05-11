package balancer

import (
	"fmt"
)

// Strategy определяет интерфейс для всех стратегий балансировки нагрузки.
// Каждая стратегия должна реализовывать метод Next для выбора следующего бэкенда.
type Strategy interface {
	// Next выбирает следующий бэкенд из пула серверов
	Next(*ServerPool) *Backend
}

// StrategyFactory создает и возвращает стратегию балансировки нагрузки
// на основе переданного имени. Поддерживаемые стратегии:
//   - "round_robin" - циклический перебор бэкендов
//   - "least_connections" - выбор бэкенда с наименьшим количеством соединений
//
// Возвращает ошибку, если переданное имя стратегии неизвестно.
func StrategyFactory(strategyName string) (Strategy, error) {
	switch strategyName {
	case "round_robin":
		return NewRoundRobinStrategy(), nil
	case "least_connections":
		return NewLeastConnectionsStrategy(), nil
	case "random":
		return NewRandomStrategy(), nil
	default:
		return nil, fmt.Errorf("unknown load balancing strategy: %s", strategyName)
	}
}
