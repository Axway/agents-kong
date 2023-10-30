.PHONY: all dep test build package

WORKSPACE ?= $$(pwd)

GO_PKG_LIST := $(shell go list ./... | grep -v /mock)

PROJECT_NAME := agents-kong

export GOFLAGS := -mod=mod
export GOPRIVATE=git.ecd.axway.org/apigov

all: clean
	@echo "Done"

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

${WORKSPACE}/discovery_agent:
	@export time=`date +%Y%m%d%H%M%S` && \
	export version=`cat version` && \
	export commit_id=`cat commit_id` && \
	export CGO_ENABLED=0 && \
	export sdk_version=`go list -m github.com/Axway/agent-sdk | awk '{print $$2}' | awk -F'-' '{print substr($$1, 2)}'` && \
	go build -v -tags static_all \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$${time}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$${version}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$${commit_id}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=$${sdk_version}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongDiscoveryAgent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Discovery Agent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildDataPlaneType=Kong'" \
		-a -o ${WORKSPACE}/bin/discovery_agent ${WORKSPACE}/cmd/discovery/main.go

build-da:dep ${WORKSPACE}/discovery_agent
	@echo "Discovery Agent build completed"

${WORKSPACE}/traceability_agent:
	@export time=`date +%Y%m%d%H%M%S` && \
	export version=`cat version` && \
	export commit_id=`cat commit_id` && \
	export CGO_ENABLED=0 && \
	export sdk_version=`go list -m github.com/Axway/agent-sdk | awk '{print $$2}' | awk -F'-' '{print substr($$1, 2)}'` && \
	go build -v -tags static_all \
		-ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=$${time}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=$${version}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=$${commit_id}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=$${sdk_version}' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongTraceabilityAgent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Traceability Agent' \
				-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildDataPlaneType=Kong'" \
		-a -o ${WORKSPACE}/bin/traceability_agent ${WORKSPACE}/cmd/traceability/main.go


build-ta: dep ${WORKSPACE}/traceability_agent
	@echo "Traceability Agent build completed"

run-disc:
	./bin/discovery

run-trace:
	./bin/traceability