# Bolt Load Balancer

A high-performance load balancer written in Go that distributes traffic across multiple backend servers using round-robin algorithm.

## Prerequisites

Before running the load balancer, you need to:

1. **Create config.yaml file** - Copy from sample_config.yaml and modify backend URLs
2. **Start backend servers** - Run your application servers on ports 8081, 8082, 8083. `If you are using different ports that's fine. Just need to update the config.yaml. For Bolt, I put 8100 as port. If you need different one. change in config.yaml and Dockerfile. `

### Sample Configuration

```yaml
server:
  port: 8100
  host: "0.0.0.0"

backends:
  - url: "http://localhost:8081"  # For non-Docker
  - url: "http://host.docker.internal:8082"  # For Docker  
  - url: "http://host.docker.internal:8083"  # For Docker

strategy: "round_robin"

health_check:
  enabled: true
  interval: "30s"
  timeout: "5s"
  path: "/health"
  expected_status: 200

logging:
  level: "info"
  format: "text"
  access_log: true
```

## Running Without Docker

### Setup and Start
I added this sample server just for test purpose. In real world or production you need not this section. Instead of this one. you will have your own backend application with multiple running instances.
```bash
# 1. Start backend servers (in separate terminals)
go run test_servers/be1.go
go run test_servers/be2.go  
go run test_servers/be3.go

# 2. Run load balancer
go run ./cmd -c config.yaml
```

### Load Testing
```bash
# Run automated test script
chmod +x bolt_test.sh
./bolt_test.sh
```

The script will automatically:
- Create configuration
- Start backend servers  
- Run load balancer
- Perform health checks
- Execute load test with 1000 requests
- Show performance statistics

## Running With Docker

### Setup and Start
```bash
# 1. Start backend servers locally (in separate terminals)
go run test_servers/be1.go # optional, use only to test the Bolt
go run test_servers/be2.go # optional, use only to test the Bolt
go run test_servers/be3.go # optional, use only to test the Bolt

# 2. Update config.yaml for Docker
# Change backend URLs to use host.docker.internal:
backends:
  - url: "http://host.docker.internal:8081"
  - url: "http://host.docker.internal:8082" 
  - url: "http://host.docker.internal:8083"

# 3. Build and run Docker container
docker build -t bolt-loadbalancer .

docker run -d \
  --name bolt-lb \
  -p 8100:8100 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  --add-host=host.docker.internal:host-gateway \
  bolt-loadbalancer:latest
```

### Load Testing with Docker
```bash
# Run automated Docker test script
chmod +x docker_test.sh
./docker_test.sh
```

The Docker test script will:
- Start backend servers locally
- Build Docker image
- Run load balancer in container
- Execute comprehensive tests
- Run load test and show results

### Manual Testing
```bash
# Check health
curl http://localhost:8100/health

# Test load balancing  
for i in {1..6}; do
  curl -s http://localhost:8100/ | head -1
done

# Load test with Apache Bench
ab -n 1000 -c 50 http://localhost:8100/
```

## Quick Commands

### Management
```bash
# Stop Docker container
docker stop bolt-lb && docker rm bolt-lb

# View logs
docker logs bolt-lb

# Kill backend processes
pkill -f "go run test_servers"
```

### Testing
```bash
# Health check
curl http://localhost:8100/health

# Status info  
curl http://localhost:8100/status

# Simple load test
for i in {1..100}; do curl -s http://localhost:8100/ >/dev/null; done
```

## Performance

The load balancer typically achieves:
- 15,000+ requests per second
- Sub-millisecond response times
- 100% success rate under normal load
- Automatic failover when backends are unavailable

Built with Go's efficient concurrency model for production-grade performance.