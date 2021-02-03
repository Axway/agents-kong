GO_VERSION=1.14.3

PROJECT_NAME := agents-kong

export GOPRIVATE=git.ecd.axway.org/apigov

tidy: go.mod
	@go mod tidy

download: tidy
	@go mod download

build-disc:
	@go build -o bin/discovery ./cmd/discovery/discovery.go

build-trace:
	@go build -o bin/traceability ./cmd/discovery/traceability.go

run-disc:
	./bin/discovery

run-trace:
	./bin/traceability

lint:
	@golangci-lint run -v

lint-fix:
	@golangci-lint run -v --fix