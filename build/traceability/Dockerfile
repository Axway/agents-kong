# Build image
# golang:1.22.4-alpine3.20 linux/amd64
FROM docker.io/golang@sha256:ace6cc3fe58d0c7b12303c57afe6d6724851152df55e08057b43990b927ad5e8 AS builder

ARG commit_id
ARG version
ARG sdk_version
ARG time
ARG CGO_ENABLED

RUN apk --no-cache update && \
  apk --no-cache add -U build-base

ENV BASEPATH=/go/src/github.com/Axway/agents-kong
ENV APP_USER axway

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
RUN addgroup -g 2500 ${APP_USER} && adduser -u 2500 -D -G ${APP_USER} ${APP_USER}
RUN chown -R ${APP_USER}:${APP_USER} /traceability_agent
USER ${APP_USER}

# alpine 3.20 linux/amd64
FROM docker.io/alpine@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0

ENV APP_USER axway

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /traceability_agent /kong_traceability_agent
COPY build/traceability/kong_traceability_agent.yml /kong_traceability_agent.yml

RUN mkdir /keys /logs && \
  chown -R axway /keys /logs && \
  find / -perm /6000 -type f -exec chmod a-s {} \; || true

USER ${APP_USER}
VOLUME ["/keys", "/logs"]
HEALTHCHECK --retries=1 CMD /kong_traceability_agent --status || exit 1
ENTRYPOINT ["/kong_traceability_agent"]