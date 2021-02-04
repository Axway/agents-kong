# Kong Traceability agent

## Overview
Kong Traceability agent does these tasks:
* Runs an HTTP Server exposing an endpoint that serves as the target for Kong's [HTTP Log Plugin](https://docs.konghq.com/hub/kong-inc/http-log/)
* Processes the request logs as they are sent by the HTTP Log plugin and builds transaction summary and transaction leg event in the format expected by Central's API Observer
* Uses libbeat to publish the events to Condor

## Prerequisites
Kong Traceability agent requires **global deployment** of the below plugins in order to generate transaction summary and transaction leg event for all Kong's proxies
* [Correlation ID Plugin](https://docs.konghq.com/hub/kong-inc/correlation-id/) used for transaction ID
* [HTTP Log Plugin](https://docs.konghq.com/hub/kong-inc/http-log/) used to get the request logs associated with Kong proxy invocation

## Build

In order to build, run 
```shell
make build-trace
```

## Configuration

Configuration can be provided via kong_traceability_agent.yml under **traceability** folder

## Run

In order to run, make sure you are in **traceability** folder and run
```shell
make run
```

