FROM beano.swf-artifactory.lab.phx.axway.int/beano-alpine-base:latest as builder

RUN mkdir -p /go/src/git.ecd.axway.org/apigov/kong_traceability_agent

WORKDIR /go/src/git.ecd.axway.org/apigov/kong_traceability_agent

COPY . .

RUN make build

RUN ls -l bin/

# Create non-root user
RUN addgroup -g 2500 axway && adduser -u 2500 -D -G axway axway
RUN chown -R axway:axway /go/src/git.ecd.axway.org/apigov/kong_traceability_agent/bin/kong_traceability_agent
USER axway

# alpine 3.12.0
FROM docker.io/alpine@sha256:a15790640a6690aa1730c38cf0a440e2aa44aaca9b0e8931a9f2b0d7cc90fd65

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/src/git.ecd.axway.org/apigov/kong_traceability_agent/kong_traceability_agent.yml /kong_traceability_agent.yml
COPY --from=builder /go/src/git.ecd.axway.org/apigov/kong_traceability_agent/bin/kong_traceability_agent /kong_traceability_agent
COPY --from=builder /go/src/git.ecd.axway.org/apigov/kong_traceability_agent/private_key.pem private_key.pem
COPY --from=builder /go/src/git.ecd.axway.org/apigov/kong_traceability_agent/public_key.pem public_key.pem

RUN mkdir /keys /data && \
    chown -R axway /data /keys /kong_traceability_agent.yml && \
    chmod go-w /kong_traceability_agent.yml

RUN find / -perm /6000 -type f -exec chmod a-s {} \; || true

USER axway

VOLUME ["/keys"]

ENTRYPOINT ["/kong_traceability_agent"]