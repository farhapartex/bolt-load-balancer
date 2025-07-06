set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
LOAD_BALANCER_PORT=8100
BACKEND1_PORT=8081
BACKEND2_PORT=8082
BACKEND3_PORT=8083
LOAD_TEST_REQUESTS=1000
LOAD_TEST_CONCURRENT=50

PIDS=()

cleanup() {
    echo -e "\n${YELLOW} Cleaning up processes...${NC}"
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            echo -e "${GREEN} Stopped process $pid${NC}"
        fi
    done
    echo -e "${GREEN} Cleanup complete${NC}"
    exit 0
}

trap cleanup INT TERM EXIT

print_header() {
    echo -e "${CYAN}"
    echo "============================================================"
    echo " BOLT LOAD BALANCER - AUTOMATED TESTING SCRIPT"
    echo "============================================================"
    echo -e "${NC}"
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

wait_for_service() {
    local url="$1"
    local name="$2"
    local max_attempts=30
    local attempt=1
    
    echo -e "${YELLOW}⏳ Waiting for $name to be ready...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" >/dev/null 2>&1; then
            echo -e "${GREEN} $name is ready!${NC}"
            return 0
        fi
        
        echo -e "${BLUE}  Attempt $attempt/$max_attempts - waiting...${NC}"
        sleep 2
        ((attempt++))
    done
    
    echo -e "${RED} $name failed to start within timeout${NC}"
    return 1
}

create_config() {
    echo -e "\n${PURPLE} Step 1: Creating configuration file...${NC}"
    
    if [ ! -f "sample_config.yaml" ]; then
        echo -e "${YELLOW}  sample_config.yaml not found. Creating default config...${NC}"
        cat > sample_config.yaml << 'EOF'
        server:
        port: 8100
        host: "0.0.0.0"
        read_timeout: "30s"
        write_timeout: "30s"
        idle_timeout: "60s"

        backends:
        - url: "http://localhost:8081"
            weight: 1
            max_fails: 3
            fail_timeout: "30s"
        - url: "http://localhost:8082"
            weight: 1
            max_fails: 3
            fail_timeout: "30s"
        - url: "http://localhost:8083"
            weight: 1
            max_fails: 3
            fail_timeout: "30s"

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

EOF
    fi
    
    cp sample_config.yaml config.yaml
    echo -e "${GREEN} Configuration file created: config.yaml${NC}"
}

start_backends() {
    echo -e "\n${PURPLE}  Step 2: Starting backend servers...${NC}"

    for i in 1 2 3; do
        if [ ! -f "test_servers/be${i}/be${i}.go" ]; then
            echo -e "${RED} Backend file test_servers/be${i}/be${i}.go not found${NC}"
            exit 1
        fi
    done

    echo -e "${BLUE} Starting Backend Server 1 (Port $BACKEND1_PORT)...${NC}"
    go run test_servers/be1/be1.go &
    PIDS+=($!)

    echo -e "${BLUE} Starting Backend Server 2 (Port $BACKEND2_PORT)...${NC}"
    go run test_servers/be2/be2.go &
    PIDS+=($!)

    echo -e "${BLUE} Starting Backend Server 3 (Port $BACKEND3_PORT)...${NC}"
    go run test_servers/be3/be3.go &
    PIDS+=($!)

    sleep 3

    wait_for_service "http://localhost:$BACKEND1_PORT/health" "Backend 1"
    wait_for_service "http://localhost:$BACKEND2_PORT/health" "Backend 2"
    wait_for_service "http://localhost:$BACKEND3_PORT/health" "Backend 3"

    echo -e "${GREEN} All backend servers are running!${NC}"
}

start_load_balancer(){
    echo -e "\n${PURPLE}  Step 3: Starting load balancer...${NC}"
    if [ ! -f "cmd/main.go" ]; then
        echo -e "${RED} Load balancer main.go not found in cmd/ directory${NC}"
        exit 1
    fi

    echo -e "${BLUE} Starting Bolt Load Balancer (Port $LOAD_BALANCER_PORT)...${NC}"
    go run ./cmd -c config.yaml &
    PIDS+=($!)

    sleep 3
    wait_for_service "http://localhost:$LOAD_BALANCER_PORT/health" "Load Balancer"

    echo -e "${GREEN} Load balancer is running!${NC}"
}

test_server_status() {
    echo -e "\n${PURPLE} Step 4: Testing server status...${NC}"
    
    echo -e "\n${CYAN} Load Balancer Health:${NC}"
    curl -s "http://localhost:$LOAD_BALANCER_PORT/health" || echo -e "${RED}❌ Health check failed${NC}"
    
    echo -e "\n\n${CYAN} Load Balancer Status:${NC}"
    curl -s "http://localhost:$LOAD_BALANCER_PORT/status" || echo -e "${RED}❌ Status check failed${NC}"
    
    echo -e "\n\n${CYAN} Backend Health Checks:${NC}"
    echo -e "${BLUE}Backend 1:${NC}"
    curl -s "http://localhost:$BACKEND1_PORT/health" || echo -e "${RED}❌ Backend 1 health check failed${NC}"
    
    echo -e "\n${BLUE}Backend 2:${NC}"
    curl -s "http://localhost:$BACKEND2_PORT/health" || echo -e "${RED}❌ Backend 2 health check failed${NC}"
    
    echo -e "\n${BLUE}Backend 3:${NC}"
    curl -s "http://localhost:$BACKEND3_PORT/health" || echo -e "${RED}❌ Backend 3 health check failed${NC}"
    
    echo -e "\n\n${CYAN} Round Robin Test (6 requests):${NC}"
    for i in {1..6}; do
        echo -e "${BLUE}Request $i:${NC}"
        curl -s "http://localhost:$LOAD_BALANCER_PORT/" | head -1 || echo -e "${RED}❌ Request $i failed${NC}"
    done
    
    echo -e "\n\n${CYAN} API Endpoint Test:${NC}"
    echo -e "${BLUE}Testing /api/v1/login:${NC}"
    curl -s "http://localhost:$LOAD_BALANCER_PORT/api/v1/login" | head -1 || echo -e "${RED}❌ API test failed${NC}"
    
    echo -e "\n${BLUE}Testing /api/v1/me:${NC}"
    curl -s "http://localhost:$LOAD_BALANCER_PORT/api/v1/me" | head -1 || echo -e "${RED}❌ API test failed${NC}"
    
    echo -e "\n${GREEN} Server status tests completed!${NC}"
}


run_load_test() {
    echo -e "\n${PURPLE} Step 5: Running load test...${NC}"
    
    if command_exists ab; then
        LOAD_TOOL="ab"
    elif command_exists curl; then
        LOAD_TOOL="curl"
    else
        echo -e "${RED} No load testing tool available (ab or curl required)${NC}"
        return 1
    fi
    
    echo -e "${CYAN} Load Test Configuration:${NC}"
    echo -e "${BLUE}   • Total Requests: $LOAD_TEST_REQUESTS${NC}"
    echo -e "${BLUE}   • Concurrent Requests: $LOAD_TEST_CONCURRENT${NC}"
    echo -e "${BLUE}   • Tool: $LOAD_TOOL${NC}"
    echo -e "${BLUE}   • Target: http://localhost:$LOAD_BALANCER_PORT/${NC}"
    
    declare -A backend_counts
    backend_counts["backend-1"]=0
    backend_counts["backend-2"]=0
    backend_counts["backend-3"]=0
    
    total_requests=0
    successful_requests=0
    failed_requests=0
    start_time=$(date +%s)
    
    if [ "$LOAD_TOOL" = "ab" ]; then
        echo -e "\n${YELLOW} Running Bolt load test...${NC}"
        
        ab_output=$(ab -n $LOAD_TEST_REQUESTS -c $LOAD_TEST_CONCURRENT "http://localhost:$LOAD_BALANCER_PORT/" 2>/dev/null)
        
        echo -e "\n${CYAN} Bolt LB Load test Results:${NC}"
        echo "$ab_output" | grep -E "(Requests per second|Time taken for tests|Transfer rate|Failed requests)"
        
        echo -e "\n${CYAN} Backend Distribution Test (10000 requests):${NC}"
        for i in {1..10000}; do
            response=$(curl -s "http://localhost:$LOAD_BALANCER_PORT/" 2>/dev/null || echo "ERROR")
            if [[ $response == *"Backend Server 1"* ]]; then
                ((backend_counts["backend-1"]++))
            elif [[ $response == *"Backend Server 2"* ]]; then
                ((backend_counts["backend-2"]++))
            elif [[ $response == *"Backend Server 3"* ]]; then
                ((backend_counts["backend-3"]++))
            fi
            
            # Progress indicator
            if [ $((i % 20)) -eq 0 ]; then
                echo -e "${BLUE}   Progress: $i/10000 requests completed${NC}"
            fi
        done
        
    else
        echo -e "\n${YELLOW} Running curl-based load test...${NC}"
        for batch in $(seq 1 $((LOAD_TEST_REQUESTS / LOAD_TEST_CONCURRENT))); do
            echo -e "${BLUE}   Batch $batch/$(( LOAD_TEST_REQUESTS / LOAD_TEST_CONCURRENT ))${NC}"
            
            for i in $(seq 1 $LOAD_TEST_CONCURRENT); do
                {
                    response=$(curl -s -w "%{http_code}" "http://localhost:$LOAD_BALANCER_PORT/" 2>/dev/null || echo "000")
                    if [[ $response == *"200" ]]; then
                        echo "SUCCESS"
                    else
                        echo "FAILED"
                    fi
                } &
            done
            
            wait
            
            ((total_requests += LOAD_TEST_CONCURRENT))
        done
    fi
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    echo -e "\n${CYAN} Load Test Statistics:${NC}"
    echo -e "${GREEN} Total Time: ${duration}s${NC}"
    echo -e "${GREEN} Requests/Second: $(( LOAD_TEST_REQUESTS / duration ))${NC}"
    
    echo -e "\n${CYAN} Backend Distribution:${NC}"
    echo -e "${GREEN}   Backend 1: ${backend_counts["backend-1"]} requests${NC}"
    echo -e "${GREEN}   Backend 2: ${backend_counts["backend-2"]} requests${NC}"
    echo -e "${GREEN}   Backend 3: ${backend_counts["backend-3"]} requests${NC}"
    
    total_dist=$((backend_counts["backend-1"] + backend_counts["backend-2"] + backend_counts["backend-3"]))
    if [ $total_dist -gt 0 ]; then
        echo -e "\n${CYAN} Distribution Percentage:${NC}"
        echo -e "${GREEN}   Backend 1: $(( backend_counts["backend-1"] * 100 / total_dist ))%${NC}"
        echo -e "${GREEN}   Backend 2: $(( backend_counts["backend-2"] * 100 / total_dist ))%${NC}"
        echo -e "${GREEN}   Backend 3: $(( backend_counts["backend-3"] * 100 / total_dist ))%${NC}"
    fi
    
    echo -e "\n${GREEN} Load test completed!${NC}"
}

show_summary() {
    echo -e "\n${CYAN}"
    echo "============================================================"
    echo " BOLT LOAD BALANCER TEST SUMMARY"
    echo "============================================================"
    echo -e "${NC}"
    
    echo -e "${GREEN} Configuration file created${NC}"
    echo -e "${GREEN} Backend servers started (3 servers)${NC}"
    echo -e "${GREEN} Load balancer started${NC}"
    echo -e "${GREEN} Server status tests completed${NC}"
    echo -e "${GREEN} Load tests completed${NC}"
    
    echo -e "\n${CYAN} Service URLs:${NC}"
    echo -e "${BLUE}   Load Balancer: http://localhost:$LOAD_BALANCER_PORT${NC}"
    echo -e "${BLUE}   Backend 1:     http://localhost:$BACKEND1_PORT${NC}"
    echo -e "${BLUE}   Backend 2:     http://localhost:$BACKEND2_PORT${NC}"
    echo -e "${BLUE}   Backend 3:     http://localhost:$BACKEND3_PORT${NC}"
    
    echo -e "\n${CYAN} Health Endpoints:${NC}"
    echo -e "${BLUE}   Health: http://localhost:$LOAD_BALANCER_PORT/health${NC}"
    echo -e "${BLUE}   Status: http://localhost:$LOAD_BALANCER_PORT/status${NC}"
    
    echo -e "\n${YELLOW}Press Ctrl+C to stop all services${NC}"
}

kill_ports() {
    echo " Killing processes on ports 8081, 8082, 8083..."
    
    for port in 8081 8082 8083; do
        pids=$(sudo lsof -t -i:$port 2>/dev/null)
        if [ -n "$pids" ]; then
            echo "   Killing processes on port $port: $pids"
            sudo kill -9 $pids
            echo "    Port $port cleared"
        else
            echo "    No processes on port $port"
        fi
    done
}

main() {
    print_header
    
    if ! command_exists go; then
        echo -e "${RED} Go is not installed or not in PATH${NC}"
        exit 1
    fi
    
    if ! command_exists curl; then
        echo -e "${RED} curl is not installed${NC}"
        exit 1
    fi
    
    create_config
    start_backends
    start_load_balancer
    test_server_status
    run_load_test
    show_summary
    kill_ports
}


main "$@"