# Getting started

The Kong agents are used to discover, provision access to, and track usages of Kong Gateway routes.

- [Getting started](#getting-started)
  - [Setup](#setup)
    - [Amplify setup](#amplify-setup)
      - [Platform - organization ID](#platform---organization-id)
      - [Platform - service account](#platform---service-account)
      - [Central - environment](#central---environment)
    - [Kong setup](#kong-setup)
      - [Kong admin API secured by Kong Gateway](#kong-admin-api-secured-by-kong-gateway)
    - [Kong agents setup](#kong-agents-setup)
      - [Additional information](#additional-information)
      - [Docker](#docker)
        - [Environment variables](#environment-variables)
        - [Deployment](#deployment)
      - [Helm](#helm)
        - [Download](#download)
        - [Create secrets](#create-secrets)
        - [Create overrides](#create-overrides)
        - [Deploy local helm chart](#deploy-local-helm-chart)
      - [Kong agent environment variables](#kong-agent-environment-variables)

## Setup

The following sections will guide you on how to setup Amplify Platform and Central then connect your Kong agents.

### Amplify setup

This section will walk you through creating an service account in Amplify Platform and an environment on Amplify Central. It will also help you find all of the required values needed to start your Kong agents.

#### Platform - organization ID

- Log into your [Amplify Platform](https://platform.axway.com) account
- Navigate to your [Organizations](https://platform.axway.com/org) page
- Note your *Organization ID*

#### Platform - service account

- Log into your [Amplify Platform](https://platform.axway.com) account
- Navigate to your [Organizations Service Accounts](https://platform.axway.com/org/client) page
- Click the *+ Service Account* button
- Set the following
  - Name: *Kong Agents*, for example
  - Description: optional
  - Tags: optional
  - Method: *Client Certificate*
  - Credentials: *Platform-generated key pair*
  - Org Roles: *Administrator* and *Central Admin*
  - Teams: Do not set any
- Download the Private Key and note its location
- Note the *Client ID* of the new service account
- Copy the *Public Key* contents and save, in the same place as the private key, naming it *public_key.pem*
- Move both of the key files to a single directory and save the path, ex: `/home/user/keys`

You now have the service account information needed for you Kong Agent installation.

#### Central - environment

- Log into Amplify Central for your Region
  - [US](https://apicentral.axway.com)
  - [EU](https://central.eu-fr.axway.com)
  - [APAC](https://central.ap-sg.axway.com/)
- On the left navigation bar select *Topology* and then *Environments*
- Click the *+ Environment* button
- Set the following
  - Environment Name: *Kong Gateway*, for example
  - Environment Type: *Custom/SDK*
  - Custom Type: *Kong*
  - Production: set the appropriate value
  - Governance: *Customer Managed*
  - Description: optional
  - Profile Image/Icon: optiona
  - Click *Next >*
- Finish up the wizard setting values as desired, on the last page click *Save*
- Note the *Logical Name* for your new environment

### Kong setup

#### Kong admin API secured by Kong Gateway

See [Kong - Securing the Admin API](https://docs.konghq.com/gateway/latest/production/running-kong/secure-admin-api/)

After following the procedures above the Kong Admin API can be secured using any authentication method that Kong provides. In this section you will learn the authentication types that the Kong agents support. As well as how to retrieve the values needed for the Kong agents.

Once the Kong admin API is secured a gateway service for it must be added to Kong and then a route configured to access the gateway service. After adding those configurations the following authentication may be added to the route. Then create a consumer, in Kong, for the agent and add credentials for that consumer. Note these credentials for later.

- Basic authentication
- API Key authentication
- OAuth2 authentication

### Kong agents setup

The Kong agents are delivered as containers, kong_discovery_agent and kong_traceability_agent. These containers can be deployed directly to a container server, such as Docker, or using the provided helm chart. In this section you will lean how to deploy the agents directly as containers or within a kubernetes cluster using the helm chart.

#### Additional information

Before beginning to deploy the agents following information will need to be gathered in addition to the details that were noted in setup.

- The full URL to connect to the Kong admin API, `KONG_ADMIN_URL`
- The host the agent will use when setting the endpoint of a discovered API, (`KONG_PROXY_HOST`)
  - The HTTP `KONG_PROXY_PORTS_HTTP` and HTTPs `KONG_PROXY_PORTS_HTTPS` ports the agent will use with the endpoint above
- The URL paths, hosted by the gateway service, to query for spec files, `KONG_SPEC_URL_PATHS`

#### Docker

##### Environment variables

In this section we will use the information gathered within the setup and additional information sections above and create two environment variable files for each agent to use. This is the minimum configuration assuming defaults for all other available settings. Note the setting below expect the use of the API Key authentication method for the [Kong admin api](#kong-admin-api-secured-by-kong-gateway).

Discovery Agent

```shell
KONG_ADMIN_URL=https://kong.url.com:8444
KONG_ADMIN_AUTH_APIKEY_HEADER="apikey"
KONG_ADMIN_AUTH_APIKEY_VALUE=123456789abcdefghijkl098765432109
KONG_PROXY_HOST=kong.proxy.endpoint.com
KONG_PROXY_PORTS_HTTP=8000
KONG_PROXY_PORTS_HTTPS=8443
KONG_SPEC_URL_PATHS=/openapi.json,/swagger.json

CENTRAL_ORGANIZATIONID=123456789
CENTRAL_AUTH_CLIENTID=kong-agents_123456789-abcd-efgh-ijkl-098765432109
CENTRAL_ENVIRONMENT=kong
CENTRAL_GRPC_ENABLED=true

AGENTFEATURES_MARKETPLACEPROVISIONING=true
```

Traceability Agent

```shell
KONG_ADMIN_URL=https://kong.url.com:8444
KONG_ADMIN_AUTH_APIKEY_HEADER="apikey"
KONG_ADMIN_AUTH_APIKEY_VALUE=123456789abcdefghijkl098765432109

CENTRAL_ORGANIZATIONID=123456789
CENTRAL_AUTH_CLIENTID=kong-agents_123456789-abcd-efgh-ijkl-098765432109
CENTRAL_ENVIRONMENT=kong
CENTRAL_GRPC_ENABLED=true

AGENTFEATURES_MARKETPLACEPROVISIONING=true
```

##### Deployment

In the following docker commands...

- `/home/user/keys` in the commands below refers to the directory where the key files were created during the last step in [Platform - service account](#platform---service-account)
- `/home/user/discovery/data:/data` and `/home/user/traceability/data:/data` are volumes that are used to store cached information to be saved outside of the container in order to persist restarts
- `discovery-agents.env` and `traceability-agents.env` are files with the various environment variable settings that are available to each agent

Kong Discovery agent

```shell
docker run -d -v /home/user/keys:/keys -v /home/user/discovery/data:/data --env-file discovery-agents.env ghcr.io/axway/kong_discovery_agent:latest
```

Kong Traceability agent

```shell
docker run -d -v /home/user/keys:/keys -v /home/user/traceability/data:/data --env-file traceability-agents.env ghcr.io/axway/kong_traceability_agent:latest
```

#### Helm

##### Download

At the current time the Kong agents helm chart is not hosted on a helm chart repository. To deploy using this helm chart you will first want to download the helm directory from your desired release tag removing the v, 0.0.1 in the sample below.

```shell
export tag=0.0.1                                                                                           # tag v0.0.1 but 'v' removed
curl -L https://github.com/Axway/agents-kong/archive/refs/tags/v${tag}.tar.gz --output kong-agents.tar.gz  # download release archive
tar xvf kong-agents.tar.gz --strip-components=2 agents-kong-${tag}/kong-agents                             # extract the helm chart in the current directory 
rm kong-agents.tar.gz                                                                                      # remove the archive
```

##### Create secrets

Platform service account key secret

kong-agent-keys.yaml

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kong-agent-keys
type: Opaque
stringData:
  private_key: |
    -----BEGIN PRIVATE KEY-----
    private
    key
    data
    -----END PRIVATE KEY-----
  public_key: |
    -----BEGIN PUBLIC KEY-----
    public
    key
    data
    -----END PUBLIC KEY-----
```

##### Create overrides

overrides.yaml

```yaml
kong:
  admin:
    url: https://kong.url.com:8444
  proxy:
    host: kong.proxy.endpoint.com
    ports:
      http: 8000
      https: 8443
  spec:
    url_paths: 
      - /openapi.json
      - /swagger.json

env:
  CENTRAL_ORGANIZATIONID: 123456789
  CENTRAL_AUTH_CLIENTID: kong-agents_123456789-abcd-efgh-ijkl-098765432109
  CENTRAL_ENVIRONMENT: kong
  CENTRAL_GRPC_ENABLED: true
  AGENTFEATURES_MARKETPLACEPROVISIONING: true
```

##### Deploy local helm chart

Assuming you are already in the desired kubernetes context and namespace, execute the following commands.

Create the secret containing the Central key files used for authentication.

```shell
kubectl apply -f kong-agent-keys.yaml
```

Install the helm chart using the created overrides file.

```shell
helm install kong-agents ./kong-agents -f overrides.yaml
```

#### Kong agent environment variables

All Kong specific environment variables available are listed below

| Name                              | Description                                                                           |
| --------------------------------- | ------------------------------------------------------------------------------------- |
| **KONG_ADMIN_URL**                | The Kong admin API URL that the agent will query against                              |
| **KONG_ADMIN_AUTH_APIKEY_HEADER** | The API Key header name the agent will use when authenticating                        |
| **KONG_ADMIN_AUTH_APIKEY_VALUE**  | The API Key value the agent will use when authenticating                              |
| **KONG_PROXY_HOST**               | The proxy endpoint that the agent will use in API Services for discovered Kong routes |
| **KONG_PROXY_PORTS_HTTP**         | The HTTP port number that the agent will set for discovered APIS                      |
| **KONG_PROXY_PORTS_HTTPS**        | The HTTPs port number that the agent will set for discovered APIS                     |
| **KONG_SPEC_URL_PATHS**           | The URL paths that the agent will query on the Gateway service for API definitions    |
