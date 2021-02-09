GO_VERSION=1.14.3

PROJECT_NAME := agents-kong

export GOPRIVATE=git.ecd.axway.org/apigov

tidy: go.mod
	@go mod tidy

download: tidy
	@go mod download

build-disc:
	@go build -o bin/discovery ./cmd/discovery/main.go

build-trace:
	@go build -o bin/traceability ./cmd/traceability/main.go

run-disc:
	./bin/discovery

run-trace:
	./bin/traceability

run:
	export DEBUG=1
	@go run cmd/discovery/main.go &
	@go run cmd/traceability/main.go


lint:
	@golangci-lint run -v

lint-fix:
	@golangci-lint run -v --fixz


cs.json: 
	@curl -s https://apicentral.axway.com/api/v1/docs -o cs.json

gen-clientreg: cs.json
	@swagger generate client --name clientreg -f $< -t pkg/clientreg -O=getProfilesForApplication
