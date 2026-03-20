.PHONY: help build run-backend run-lb test clean generate-config

help:
	@echo "Load Balancer with Sticky Sessions - Development Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build                              - Build all binaries"
	@echo "  make run-backend PORT=8081 NAME=backend-1  - Start a backend server"
	@echo "  make run-lb                             - Start the load balancer"
	@echo "  make test                               - Run tests"
	@echo "  make clean                              - Clean build artifacts"
	@echo "  make generate-config                    - Generate default config file"
	@echo "  make demo                               - Show demo instructions"
	@echo ""

build:
	go build -o bin/loadbalancer cmd/server/main.go
	go build -o bin/backend cmd/backend/main.go

run-backend:
	go run cmd/backend/main.go -port $(or $(PORT),8081) -name $(or $(NAME),backend-1)

run-lb:
	go run cmd/server/main.go -config configs/config.toml

run-lb-flags:
	go run cmd/server/main.go \
		-port 8080 \
		-backends "http://localhost:8081,http://localhost:8082,http://localhost:8083" \
		-session-ttl 30m \
		-health-interval 10s \
		-health-timeout 2s

test:
	go test -v ./...

clean:
	rm -rf bin/

generate-config:
	go run cmd/server/main.go -generate-config

demo:
	@echo "Starting demo environment..."
	@echo "This will start 3 backend servers and 1 load balancer"
	@echo ""
	@echo "In Terminal 1: make run-backend PORT=8081 NAME=backend-1"
	@echo "In Terminal 2: make run-backend PORT=8082 NAME=backend-2"
	@echo "In Terminal 3: make run-backend PORT=8083 NAME=backend-3"
	@echo "In Terminal 4: make run-lb"
	@echo ""
	@echo "Then test with:"
	@echo "  curl -c cookies.txt http://localhost:8080/"
	@echo "  curl -b cookies.txt http://localhost:8080/"
	@echo ""
	@echo "Check metrics at: http://localhost:9090/metrics"

.PHONY: config-example
config-example:
	@echo "# Example: Weighted Round-Robin Configuration"
	@cat configs/config.toml