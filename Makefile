GO_VERSION=1.14.3

PROJECT_NAME := agents-kong

export GOPRIVATE=git.ecd.axway.org/apigov

tidy: go.mod
	@go mod tidy

download: tidy
	@go mod download

build:
	@go build -o bin/agents-kong ./cmd/main.go

run-discovery:
	./bin/agents-kong apic_discovery_agent

run-traceability:
	./bin/agents-kong apic_traceability_agent

lint:
	@golangci-lint run -v

lint-fix:
	@golangci-lint run -v --fix