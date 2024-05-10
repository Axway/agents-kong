.PHONY: all dep test build package

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /mock)
PROJECT_NAME := agents-kong
TIME := $(shell date +%Y%m%d%H%M%S)
VERSION := $(shell git tag -l --sort='version:refname' | grep -Eo '[0-9]{1,}\.[0-9]{1,}\.[0-9]{1,}$$' | tail -1)
CGO_ENABLED := 0
COMMIT_ID := $(shell git rev-parse --short HEAD)
SDK_VERSION := $(shell go list -m github.com/Axway/agent-sdk | awk '{print $$2}' | awk -F'-' '{print substr($$1, 2)}')

export GOFLAGS := -mod=mod
export GOPRIVATE=git.ecd.axway.org/apigov

all: clean
	@echo "Done"

test: dep
	@go vet ${GO_PKG_LIST}
	@go test -race -v -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST}

test-s: dep
	@go vet ${GO_PKG_LIST}
	@go test -race -short -coverprofile=${WORKSPACE}/gocoverage.out -count=1 ${GO_PKG_LIST}

clean:
	@rm -rf ./bin/
	@mkdir -p ./bin
	@echo "Clean complete"

resolve-dependencies:
	@echo "Resolving go package dependencies"
	@go mod tidy
	@echo "Package dependencies completed"

dep: resolve-dependencies

dep-check:
	@go mod verify

dep-version:
	@echo "$(sdk)"
	@export version=$(sdk) && make update-sdk && make dep

dep-sdk:
	@make sdk=main dep-version

update-sdk:
	@echo "Updating SDK dependencies"
	@echo "version is $(version)"
	@export GOFLAGS="" && go mod edit -require "github.com/Axway/agent-sdk@${version}"


${WORKSPACE}/discovery_agent:
	@go build -v -tags static_all \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$(TIME)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$(VERSION)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$(COMMIT_ID)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=$(SDK_VERSION)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongDiscoveryAgent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Discovery Agent'" \
		-a -o ${WORKSPACE}/bin/discovery_agent ${WORKSPACE}/pkg/discovery/main/agent.go

build-da: dep ${WORKSPACE}/discovery_agent
	@echo "Discovery Agent build completed"

${WORKSPACE}/traceability_agent:
	go build -v -tags static_all \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$(TIME)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$(VERSION)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$(COMMIT_ID)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=$(SDK_VERSION)' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongTraceabilityAgent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Traceability Agent'" \
		-a -o ${WORKSPACE}/bin/traceability_agent ${WORKSPACE}/pkg/traceability/main/agent.go

build-ta: dep ${WORKSPACE}/traceability_agent
	@echo "Traceability Agent build completed"

build: build-da build-ta

docker-da:
	docker build --build-arg commit_id=$(COMMIT_ID) --build-arg time=$(TIME) --build-arg CGO_ENABLED=$(CGO_ENABLED) --build-arg version=$(VERSION) --build-arg sdk_version=$(SDK_VERSION) -t kong-discovery-agent:latest -f ${WORKSPACE}/build/discovery/Dockerfile .
	@echo "DA Docker build complete"

docker-ta:
	docker build --build-arg commit_id=$(COMMIT_ID) --build-arg time=$(TIME) --build-arg CGO_ENABLED=$(CGO_ENABLED) --build-arg version=$(VERSION) --build-arg sdk_version=$(SDK_VERSION) -t kong-traceability-agent:latest -f ${WORKSPACE}/build/traceability/Dockerfile .
	@echo "TA Docker build complete"

docker: docker-da docker-ta
	@echo "Docker build complete"
