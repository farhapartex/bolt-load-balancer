package loadbalancer

import (
	"errors"
	"sync/atomic"
)

type Algorithm interface {
	NextBackend(backends []*Backend) *Backend
	Name() string
}

type RoundRobinAlgorithm struct {
	current uint64
}

func NewRoundRobinAlgorithm() *RoundRobinAlgorithm {
	return &RoundRobinAlgorithm{current: 0}
}

func (rr *RoundRobinAlgorithm) NextBackend(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	healthyBackends := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		if backend.IsHealthy() {
			healthyBackends = append(healthyBackends, backend)
		}
	}
	if len(healthyBackends) == 0 {
		return nil
	}
	next := atomic.AddUint64(&rr.current, 1)
	index := (next - 1) % uint64(len(healthyBackends))
	return healthyBackends[index]
}

func (rr *RoundRobinAlgorithm) Name() string {
	return "round_robin"
}

type AlgorithmFactory struct{}

func NewAlgorithmFactory() *AlgorithmFactory {
	return &AlgorithmFactory{}
}

func (af *AlgorithmFactory) CreateAlgorithm(strategy string) (Algorithm, error) {
	// More algorithm will be added here in future
	switch strategy {
	case "round_robin":
		return NewRoundRobinAlgorithm(), nil
	default:
		return nil, errors.New("unsupported load balancing strategy: " + strategy)
	}
}

func (af *AlgorithmFactory) GetSupportedAlgorithms() []string {
	return []string{
		"round_robin",
	}
}
