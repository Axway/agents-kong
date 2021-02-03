# Kong Traceability agent

## Overview
Kong Traceability agent does these tasks:
* Runs an HTTP Server exposing an endpoint that serves as the target for Kong's [HTTP Log Plugin](https://docs.konghq.com/hub/kong-inc/http-log/)
* Processes the request logs as they are sent by the HTTP Log plugin and builds transaction summary and transaction events in the format expected by Central's API Observer
* Uses libbeat to publish the events to Condor

## Build

In order to build, navigate to **traceability** folder and run 
```shell
make build
```

## Configuration

Configuration can be provided via kong_traceability_agent.yml under **traceability** folder

## Run

In order to run, make sure you are in **traceability** folder and run
```shell
make run
```

