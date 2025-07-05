package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
	"github.com/farhapartex/bolt-load-balancer/internal/health"
	"github.com/farhapartex/bolt-load-balancer/internal/loadbalancer"
	"github.com/farhapartex/bolt-load-balancer/internal/logger"
)

type LB struct {
	config        *config.Config
	backendPool   *loadbalancer.BackendPool
	algorithm     loadbalancer.Algorithm
	healthChecker *health.HealthChecker
	logger        *logger.Logger
	httpServer    *http.Server
}

func (lb *LB) handleHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	healthyBackends := lb.backendPool.HealthySize()
	totalBackends := lb.backendPool.Size()

	if healthyBackends == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "UNHEALTHY: 0/%d backends available", totalBackends)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "HEALTHY: %d/%d backends available", healthyBackends, totalBackends)
}

func (lb *LB) handleStatusEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := lb.healthChecker.GetHealthStatus(lb.backendPool)
	status["load_balancer"] = map[string]interface{}{
		"version":   "0.1.0",
		"algorithm": lb.algorithm.Name(),
		"uptime":    time.Since(time.Now()).String(), // Will be implemented properly in future versions
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "ok",
		"healthy_backends": %d,
		"total_backends": %d,
		"algorithm": "%s"
	}`, lb.backendPool.HealthySize(), lb.backendPool.Size(), lb.algorithm.Name())
}

func (lb *LB) logRequest(r *http.Request, statusCode int, duration time.Duration) {
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "-"
	}

	lb.logger.LogRequest(r.Method, r.URL.Path, r.RemoteAddr, userAgent, statusCode, duration)
}

func (lb *LB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start_time := time.Now()

	if r.URL.Path == "/health" {
		lb.handleHealthEndpoint(w, r)
	}

	if r.URL.Path == "/status" {
		lb.handleStatusEndpoint(w, r)
	}

	backend := lb.algorithm.NextBackend(lb.backendPool.GetBackends())
	if backend == nil {
		lb.logger.Warn("No healthy backends available")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		lb.logRequest(r, http.StatusServiceUnavailable, time.Since(start_time))
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend.URL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		lb.logger.LogBackendRequest(backend.URL.String(), r.Method, r.URL.Path, 0, time.Since(start_time), err)
		backend.MarkUnhealthy()
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		duration := time.Since(start_time)
		lb.logger.LogBackendRequest(backend.URL.String(), r.Method, r.URL.Path, resp.StatusCode, duration, nil)

		if resp.StatusCode < 500 {
			backend.MarkHealthy()
		} else {
			backend.MarkUnhealthy()
		}

		return nil
	}
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	r.Header.Set("X-Forwarded-Proto", "http")
	if r.Header.Get("X-Real-IP") == "" {
		r.Header.Set("X-Real-IP", r.RemoteAddr)
	}

	// Forward the request to the backend
	proxy.ServeHTTP(w, r)
	lb.logRequest(r, http.StatusOK, time.Since(start_time))
}

func (lb *LB) Start() error {
	lb.logger.Infof("Starting load balancer on %s", lb.httpServer.Addr)
	lb.logger.Infof("Using %s algorithm with %d backends", lb.algorithm.Name(), lb.backendPool.Size())

	lb.healthChecker.Start(lb.backendPool)
	lb.logger.Info("Health checker started")
	return lb.httpServer.ListenAndServe()
}

func (lb *LB) Stop(ctx context.Context) error {
	lb.logger.Info("Shutting down load balancer...")
	lb.healthChecker.Stop()
	lb.logger.Info("Health checker stopped")
	return lb.httpServer.Shutdown(ctx)
}

func New(conf *config.Config) (*LB, error) {
	backendPool := loadbalancer.NewBackendPool()

	for _, be_config := range conf.Backends {
		backend, err := loadbalancer.NewBackend(
			be_config.URL,
			be_config.Weight,
			be_config.MaxFails,
			be_config.FailTimeout,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create backend %s: %w", be_config.URL, err)
		}
		backendPool.AddBackend(backend)
	}
	factory := loadbalancer.NewAlgorithmFactory()
	algorithm, err := factory.CreateAlgorithm(conf.Strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create algorithm: %w", err)
	}

	healthChecker := health.NewHealthChecker(conf.HealthCheck)
	lgr := logger.NewLogger(conf.Logging)

	load_balance := &LB{
		config:        conf,
		backendPool:   backendPool,
		algorithm:     algorithm,
		healthChecker: healthChecker,
		logger:        lgr,
	}
	load_balance.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port),
		Handler:      load_balance,
		ReadTimeout:  conf.Server.ReadTimeout,
		WriteTimeout: conf.Server.WriteTimeout,
		IdleTimeout:  conf.Server.IdleTimeout,
	}

	return load_balance, nil
}
