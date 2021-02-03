GO_VERSION=1.14.3

PROJECT_NAME := agents-kong

export GOPRIVATE=git.ecd.axway.org/apigov

tidy: go.mod
	@go mod tidy

download: tidy
	@go mod download

build:
	@go build -o bin/kong_traceability_agent main.go

run:
	./bin/kong_traceability_agent

lint:
	@golangci-lint run -v

lint-fix:
	@golangci-lint run -v --fix