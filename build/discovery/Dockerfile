# Build image
# golang:1.22.3-alpine3.20 linux/amd64
FROM docker.io/golang@sha256:421bc7f4b90d042c56282bb894451108f8ab886687e1b73abaefad31ab10a14d AS builder

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

# alpine 3.20 linux/amd64
FROM docker.io/alpine@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd

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