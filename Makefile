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
	./bin/agents-kong kong_discovery_agent run

run-traceability:
	./bin/agents-kong kong_traceability_agent run

lint:
	@golangci-lint run -v

lint-fix:
	@golangci-lint run -v --fix