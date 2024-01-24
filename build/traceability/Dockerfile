# Build image
# golang:1.21.6-alpine3.19 linux/amd64
FROM docker.io/golang@sha256:2523a6f68a0f515fe251aad40b18545155135ca6a5b2e61da8254df9153e3648 AS builder

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
  -X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentName=KongTraceabilityAgent' \
	-X 'github.com/Axway/agent-sdk/pkg/cmd.BuildAgentDescription=Kong Traceability Agent'" \
  -a -o /traceability_agent ${BASEPATH}/pkg/traceability/main/agent.go
  
# Create non-root user
RUN addgroup -g 2500 axway && adduser -u 2500 -D -G axway axway
RUN chown -R axway:axway /traceability_agent
USER axway

# alpine 3.19 linux/amd64 
FROM docker.io/alpine@sha256:13b7e62e8df80264dbb747995705a986aa530415763a6c58f84a3ca8af9a5bcd

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /traceability_agent /kong_traceability_agent
COPY build/traceability/kong_traceability_agent.yml /kong_traceability_agent.yml

RUN mkdir /keys && \
  chown -R axway /keys && \
  apk --no-cache add openssl libssl3 libcrypto3 musl musl-utils libc6-compat busybox curl && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

USER axway

VOLUME ["/keys"]

HEALTHCHECK --retries=1 CMD /kong_traceability_agent --status || exit 1

ENTRYPOINT ["/kong_traceability_agent"]