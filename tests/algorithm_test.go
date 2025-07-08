package tests

import (
	"net/url"
	"testing"

	"github.com/farhapartex/bolt-load-balancer/internal/loadbalancer"
)

func TestRoundRobinAlgorithm(t *testing.T) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()

	if algorithm.Name() != "round_robin" {
		t.Errorf("Expected algorithm name 'round_robin', got %s", algorithm.Name())
	}

	backends := createTestBackends(t, []string{
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
	})

	for _, backend := range backends {
		backend.MarkHealthy()
	}

	selectedBackends := make([]string, 6)
	for i := 0; i < 6; i++ {
		selected := algorithm.NextBackend(backends)
		if selected == nil {
			t.Fatalf("NextBackend returned nil on iteration %d", i)
		}
		selectedBackends[i] = selected.URL.String()
	}

	expected := []string{
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
	}

	for i, expected := range expected {
		if selectedBackends[i] != expected {
			t.Errorf("Round-robin iteration %d: expected %s, got %s", i, expected, selectedBackends[i])
		}
	}
}

func TestRoundRobinWithUnhealthyBackends(t *testing.T) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()

	backends := createTestBackends(t, []string{
		"http://backend1:8081",
		"http://backend2:8082",
		"http://backend3:8083",
	})

	backends[0].MarkHealthy()
	backends[2].MarkHealthy()
	backends[1].MarkUnhealthy()

	selectedURLs := make(map[string]int)
	for i := 0; i < 10; i++ {
		selected := algorithm.NextBackend(backends)
		if selected == nil {
			t.Fatalf("NextBackend returned nil when healthy backends available")
		}
		selectedURLs[selected.URL.String()]++
	}

	if _, exists := selectedURLs["http://backend2:8080"]; exists {
		t.Error("Unhealthy backend2 was selected")
	}

	if selectedURLs["http://backend1:8081"] == 0 {
		t.Error("Healthy backend1 was never selected")
	}

	if selectedURLs["http://backend3:8083"] == 0 {
		t.Error("Healthy backend3 was never selected")
	}

	backend1Count := selectedURLs["http://backend1:8080"]
	backend3Count := selectedURLs["http://backend3:8080"]

	if backend1Count != backend3Count {
		t.Errorf("Uneven distribution: backend1=%d, backend3=%d", backend1Count, backend3Count)
	}
}

func TestRoundRobinEmptyBackends(t *testing.T) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()

	selected := algorithm.NextBackend([]*loadbalancer.Backend{})
	if selected != nil {
		t.Errorf("Expected nil for empty backends slice, got %s", selected.URL.String())
	}
}

func TestRoundRobinConcurrency(t *testing.T) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()

	backends := createTestBackends(t, []string{
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
	})

	for _, backend := range backends {
		backend.MarkHealthy()
	}

	results := make(chan string, 100)

	for i := 0; i < 100; i++ {
		go func() {
			selected := algorithm.NextBackend(backends)
			if selected != nil {
				results <- selected.URL.String()
			} else {
				results <- "nil"
			}
		}()
	}

	selectedCount := make(map[string]int)
	for i := 0; i < 100; i++ {
		result := <-results
		if result == "nil" {
			t.Error("Got nil result in concurrent test")
		}
		selectedCount[result]++
	}

	expectedCount := 100 / 3 // ~33 each
	tolerance := 5

	for url, count := range selectedCount {
		if count < expectedCount-tolerance || count > expectedCount+tolerance {
			t.Errorf("Backend %s selected %d times, expected around %d", url, count, expectedCount)
		}
	}
}

func TestAlgorithmFactory(t *testing.T) {
	factory := loadbalancer.NewAlgorithmFactory()

	supportedAlgorithms := factory.GetSupportedAlgorithms()
	if len(supportedAlgorithms) == 0 {
		t.Error("Factory should support at least one algorithm")
	}

	found := false
	for _, alg := range supportedAlgorithms {
		if alg == "round_robin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Factory should support 'round_robin' algorithm")
	}

	algorithm, err := factory.CreateAlgorithm("round_robin")
	if err != nil {
		t.Errorf("Failed to create round_robin algorithm: %v", err)
	}

	if algorithm == nil {
		t.Error("Created algorithm should not be nil")
	}

	if algorithm.Name() != "round_robin" {
		t.Errorf("Created algorithm name should be 'round_robin', got %s", algorithm.Name())
	}

	_, err = factory.CreateAlgorithm("unsupported_algorithm")
	if err == nil {
		t.Error("Should return error for unsupported algorithm")
	}

	expectedError := "unsupported load balancing strategy"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAlgorithmFactoryNilCreation(t *testing.T) {
	factory := loadbalancer.NewAlgorithmFactory()

	_, err := factory.CreateAlgorithm("")
	if err == nil {
		t.Error("Should return error for empty algorithm name")
	}

	_, err = factory.CreateAlgorithm("   ")
	if err == nil {
		t.Error("Should return error for whitespace algorithm name")
	}
}

func createTestBackends(t *testing.T, urls []string) []*loadbalancer.Backend {
	var backends []*loadbalancer.Backend

	for _, urlStr := range urls {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			t.Fatalf("Failed to parse URL %s: %v", urlStr, err)
		}

		backend := &loadbalancer.Backend{
			URL:    parsedURL,
			Weight: 1,
		}
		backends = append(backends, backend)
	}

	return backends
}

func BenchmarkRoundRobinNextBackend(b *testing.B) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()
	backends := createTestBackends(&testing.T{}, []string{
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
	})

	// Mark all as healthy
	for _, backend := range backends {
		backend.MarkHealthy()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		algorithm.NextBackend(backends)
	}
}

func BenchmarkRoundRobinConcurrent(b *testing.B) {
	algorithm := loadbalancer.NewRoundRobinAlgorithm()
	backends := createTestBackends(&testing.T{}, []string{
		"http://backend1:8080",
		"http://backend2:8080",
		"http://backend3:8080",
	})

	for _, backend := range backends {
		backend.MarkHealthy()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			algorithm.NextBackend(backends)
		}
	})
}
