package loadbalancer

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

type BackendStatus int

const (
	// StatusHealthy indicates the backend is responding normally
	StatusHealthy BackendStatus = iota

	// StatusUnhealthy indicates the backend is not responding or failing
	StatusUnhealthy

	// StatusUnknown indicates the backend status hasn't been checked yet
	StatusUnknown
)

func (s BackendStatus) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

type Backend struct {
	URL             *url.URL
	Weight          int
	MaxFails        int
	FailTimeout     time.Duration
	Status          BackendStatus
	FailCount       int
	LastFailTime    time.Time
	LastHealthCheck time.Time

	mutex sync.RWMutex
}

func (b *Backend) IsHealthy() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.Status == StatusUnhealthy {
		if time.Since(b.LastFailTime) > b.FailTimeout {
			return true
		}
		return false
	}

	return b.Status == StatusHealthy
}

func (b *Backend) MarkHealthy() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.Status = StatusHealthy
	b.FailCount = 0
	b.LastHealthCheck = time.Now()
}

func (b *Backend) MarkUnhealthy() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.FailCount++
	b.LastFailTime = time.Now()

	if b.FailCount >= b.MaxFails {
		b.Status = StatusUnhealthy
	}
}

func (b *Backend) GetStatus() BackendStatus {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.Status
}

func (b *Backend) GetFailCount() int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.FailCount
}

func (b *Backend) GetWeight() int {
	return b.Weight
}

func (b *Backend) DataReprensation() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return fmt.Sprintf("Backend{URL: %s, Status: %s, Weight: %d, Fails: %d/%d}",
		b.URL.String(), b.Status.String(), b.Weight, b.FailCount, b.MaxFails)
}

func NewBackend(rawURL string, weight int, maxFails int, failTimeout time.Duration) (*Backend, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	return &Backend{
		URL:         parsedURL,
		Weight:      weight,
		MaxFails:    maxFails,
		FailTimeout: failTimeout,
		Status:      StatusUnknown,
		FailCount:   0,
	}, nil
}

// BackendPool manages a collection of backend servers.
type BackendPool struct {
	backends []*Backend
	mutex    sync.RWMutex
}

func (bp *BackendPool) AddBackend(backend *Backend) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.backends = append(bp.backends, backend)
}

func (bp *BackendPool) GetBackends() []*Backend {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	backends := make([]*Backend, len(bp.backends))
	copy(backends, bp.backends)
	return backends
}

func (bp *BackendPool) GetHealthyBackends() []*Backend {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	healthy := make([]*Backend, 0)
	for _, backend := range bp.backends {
		if backend.IsHealthy() {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

func (bp *BackendPool) Size() int {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return len(bp.backends)
}

func (bp *BackendPool) HealthySize() int {
	return len(bp.GetHealthyBackends())
}

func NewBackendPool() *BackendPool {
	return &BackendPool{
		backends: make([]*Backend, 0),
	}
}
