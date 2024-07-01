# Build image
# golang:1.22.4-alpine3.20 linux/amd64
FROM docker.io/golang@sha256:ace6cc3fe58d0c7b12303c57afe6d6724851152df55e08057b43990b927ad5e8 AS builder

ARG commit_id
ARG version
ARG sdk_version
ARG time
ARG CGO_ENABLED

ENV GOFLAGS "-mod=mod"
ENV GOWORK "off"

RUN apk --no-cache update && \
  apk --no-cache add -U build-base

ENV BASEPATH=/go/src/github.com/Axway/agents-kong
RUN mkdir -p ${BASEPATH}
WORKDIR ${BASEPATH}

# Copy only the go files to download all dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy necessary files and build
COPY pkg ./pkg/

RUN --mount=type=cache,target=/root/.cache/go-build \
  go build -tags static_all \
  -ldflags="-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildTime=${time}' \
  -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildVersion=${version}' \
  -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildCommitSha=${commit_id}' \
  -X 'github.com/Axway/agent-sdk/pkg/cmd.SDKBuildVersion=${sdk_version}' \
  -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongDiscoveryAgent' \
	-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Discovery Agent'" \
  -a -o /discovery_agent ${BASEPATH}/pkg/discovery/main/agent.go

# Create non-root user
RUN addgroup -g 2500 axway && adduser -u 2500 -D -G axway axway
RUN chown -R axway:axway /discovery_agent
USER axway

# alpine 3.20.1 linux/amd64
FROM docker.io/alpine@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /discovery_agent /kong_discovery_agent

RUN mkdir /keys && \
  chown -R axway /keys && \
  touch /kong_discovery_agent.yml && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

USER axway

VOLUME ["/keys"]

HEALTHCHECK --retries=1 CMD /kong_discovery_agent --status || exit 1

ENTRYPOINT ["/kong_discovery_agent"]