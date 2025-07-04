package loadbalancer

import "sync/atomic"

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
