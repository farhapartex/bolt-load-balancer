package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
	"github.com/farhapartex/bolt-load-balancer/internal/loadbalancer"
)

type HealthChecker struct {
	config     config.HealthCheckConfig
	httpClient *http.Client
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func (hc *HealthChecker) checkBackend(backend *loadbalancer.Backend) {
	healthURL := fmt.Sprintf("%s%s", backend.URL.String(), hc.config.Path)

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		backend.MarkUnhealthy()
		return
	}

	req.Header.Set("User-Agent", "BoltLoadBalancer/0.1.0 HealthChecker")
	req.Header.Set("Accept", "*/*")

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		backend.MarkUnhealthy()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == hc.config.ExpectedStatus {
		backend.MarkHealthy()
	} else {
		backend.MarkUnhealthy()
	}
}

func (hc *HealthChecker) checkAllBackends(backendPool *loadbalancer.BackendPool) {
	backends := backendPool.GetBackends()
	var wg sync.WaitGroup

	for _, backend := range backends {
		wg.Add(1)
		go func(b *loadbalancer.Backend) {
			defer wg.Done()
			hc.checkBackend(b)
		}(backend)
	}

	wg.Wait()
}

func (hc *HealthChecker) runHealthChecks(backendPool *loadbalancer.BackendPool) {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	hc.checkAllBackends(backendPool)
	for {
		select {
		case <-ticker.C:
			hc.checkAllBackends(backendPool)
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *HealthChecker) Start(backendPool *loadbalancer.BackendPool) {
	if !hc.config.Enabled {
		return
	}

	hc.wg.Add(1)
	go hc.runHealthChecks(backendPool)
}

func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
}
