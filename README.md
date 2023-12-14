# Getting started

The Kong agents are used to discover, provision access to, and track usages of Kong Gateway routes.

- [Getting started](#getting-started)
  - [Discovery process](#discovery-process)
  - [Provisioning process](#provisioning-process)
    - [Marketplace application](#marketplace-application)
    - [Access request](#access-request)
    - [Credential](#credential)
  - [Traceability process](#traceability-process)
  - [Environment variables](#environment-variables)
  - [Setup](#setup)
    - [Amplify setup](#amplify-setup)
      - [Platform - organization ID](#platform---organization-id)
      - [Platform - service account](#platform---service-account)
      - [Central - environment](#central---environment)
    - [Kong setup](#kong-setup)
      - [Kong admin API secured by Kong Gateway](#kong-admin-api-secured-by-kong-gateway)
      - [Specification discovery methods](#specification-discovery-methods)
        - [Local specification path](#local-specification-path)
        - [Filtering gateway services](#filtering-gateway-services)
        - [URL specification paths](#url-specification-paths)
        - [Kong Dev Portal](#kong-dev-portal)
      - [HTTP Log plugin](#http-log-plugin)
  - [Kong agents deployment](#kong-agents-deployment)
    - [Additional information](#additional-information)
    - [Docker](#docker)
      - [Environment variable files](#environment-variable-files)
      - [Deployment](#deployment)
    - [Helm](#helm)
      - [Traceability agent stateful set](#traceability-agent-stateful-set)
      - [Create secrets](#create-secrets)
      - [Create volume, local specification files only](#create-volume-local-specification-files-only)
        - [ConfigMap](#configmap)
        - [AWS S3 Synchronization](#aws-s3-synchronization)
      - [Create overrides](#create-overrides)
      - [Deploy helm chart](#deploy-helm-chart)

## Discovery process

On startup the Kong discovery agent first validates that it is able to connect to all required services. Once connected to Kong the agent begins looking at the Plugins configured, more specifically for the ACL. The default option is to require having it. This can be changed from the config by disabling this check. By having the check disabled, it is assumed that access is allowed for everyone. Then the agent will determine, from the plugins, which credential types the Kong Gateway has configured and create the Central representation of those types.

After that initial startup process the discovery agent begins running its main discovery loop. In this loop the agent first gets a list of all Gateway Services. With each service the agent looks for all configured routes. The agent then looks to gather the specification file, see [Specification discovery methods](#specification-discovery-methods), if found the process continues. Using the route the agent checks for plugins to determine the types of credentials to associate with it. After gathering all of this information the agent creates a new API service with the specification file and linking the appropriate credentials. The endpoints associated to the API service are constructed using the **KONG_PROXY_HOST**, **KONG_PROXY_PORTS_HTTP**, and **KONG_PROXY_PORTS_HTTPS** settings.

## Provisioning process

As described in the [Discovery process](#discovery-process) section the Kong agent creates all supported credential types on Central at startup. Once API services are published they can be made into Assets and Products via Central itself. The Products can then be published to the Marketplace for consumption. In order to receive access to the service a user must first request access to it and the Kong agent provisioning process will execute based off of that request.

### Marketplace application

A Marketplace application is created by a Marketplace user. When a resource within the Kong environment is added to that application Central will create a ManagedApplication resource that the agent will execute off of. This ManagedApplication resource event is captured by the Kong agent and the agent creates a Kong consumer. In addition to the creation of the Consumer the agent adds an ACL Group ID to the Consumer, to be used by the Access Request.

### Access request

(Note: if the ACL plugin is not required, access request is skipped altogether). When a Marketplace user requests access to a resource, within the Kong environment, Central will create an AccessRequest resource in the same Kong environment. The agent receives this event and makes several changes within Kong. First the agent will add, or update, an ACL configuration on the Route being requested. This ACL will allow the Group ID created during the handling of the [Marketplace application](#marketplace-application) access to the route. Additionally, if a quota for this route has been set in Central in the product being handled the agent will add a Rate limiting plugin to reflect the quota that was set in Central for that product. Note: Quotas in Central can have a Weekly amount, this is not supported by Kong and the agent will reject the Access Request.

### Credential

Finally, when a Marketplace user requests a credential, within the Kong environment, Central will create a Credential resource in the same Kong environment. The agent receives this event and creates the proper credential type for the Consumer that the [Marketplace application](#marketplace-application) handling created. After successfully creating this credential the necessary details are returned back to the Central to be viewed and used by the Marketplace user.

## Traceability process

On startup the Kong traceability agent first validates that it is able to connect to all required services. Once validation is complete the agent begins listening for log events to be sent to it. The agent receives these events and iterates through them to determine if any of the events should be sampled. If it is to be sampled the agent creates a transaction summary and leg sending that the Amplify Central. Regardless of the event being set for sampling the agent will update the proper API Metric and Usage details to be sent to Amplify Central on the interval configured. See [Usage](https://docs.axway.com/bundle/amplify-central/page/docs/connect_manage_environ/connected_agent_common_reference/traceability_usage/index.html). Note: if the ACL plugin is not required, the traceability agent cannot associate API traffic with a consumer application.

## Environment variables

All Kong specific environment variables available are listed below

| Name                                   | Description                                                                                                                                                                                                                                        |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Discovery Agent Variables              |                                                                                                                                                                                                                                                    |
| **KONG_ACL_DISABLE**                   | Set to true to disable the check for a globally enabled ACL plugin on Kong. False by default.                                                                                                                                                      |
| **KONG_ADMIN_URL**                     | The Kong admin API URL that the agent will query against                                                                                                                                                                                           |
| **KONG_ADMIN_AUTH_APIKEY_HEADER**      | The API Key header name the agent will use when authenticating                                                                                                                                                                                     |
| **KONG_ADMIN_AUTH_APIKEY_VALUE**       | The API Key value the agent will use when authenticating                                                                                                                                                                                           |
| **KONG_ADMIN_AUTH_BASICAUTH_USERNAME** | The HTTP Basic username that the agent will use when authenticating                                                                                                                                                                                |
| **KONG_ADMIN_AUTH_BASICAUTH_PASSWORD** | The HTTP Basic password that the agent will use when authenticating                                                                                                                                                                                |
| **KONG_ADMIN_SSL_NEXTPROTOS**          | An array of strings. It is a list of supported application level protocols, in order of preference, based on the ALPN protocol list. Allowed values are: h2, http/1.0, http/1.1, h2c                                                               |
| **KONG_ADMIN_SSL_INSECURESKIPVERIFY**  | Controls whether a client verifies the server’s certificate chain and host name. If true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks |
| **KONG_ADMIN_SSL_CIPHERSUITES**        | An array of strings. It is a list of supported cipher suites for TLS versions up to TLS 1.2. If CipherSuites is nil, a default list of secure cipher suites is used, with a preference order based on hardware performance                         |
| **KONG_ADMIN_SSL_MAXVERSION**          | String value for the maximum SSL/TLS version that is acceptable. If empty, then the maximum version supported by this package is used, which is currently TLS 1.3. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                              |
| **KONG_ADMIN_SSL_MINVERSION**          | String value for the minimum SSL/TLS version that is acceptable. If empty TLS 1.2 is taken as the minimum. Allowed values are: TLS1.0, TLS1.1, TLS1.2, TLS1.3                                                                                      |
| **KONG_PROXY_HOST**                    | The proxy host that the agent will use in API Services when the Kong route does not specify hosts                                                                                                                                                  |
| **KONG_PROXY_PORTS_HTTP_VALUE**        | The HTTP port value that the agent will set for discovered APIS                                                                                                                                                                                    |
| **KONG_PROXY_PORTS_HTTPS_VALUE**       | The HTTPs port value that the agent will set for discovered APIS                                                                                                                                                                                   |
| **KONG_PROXY_PORTS_HTTP_DISABLE**      | Set to true if the agent should ignore routes that serve over HTTP                                                                                                                                                                                 |
| **KONG_PROXY_PORTS_HTTPS_DISABLE**     | Set to true if the agent should ignore routes that serve over HTTPs                                                                                                                                                                                |
| **KONG_PROXY_BASEPATH**                | The proxy base path that will be added between the proxy host and Kong route path when building endpoints                                                                                                                                          |
| **KONG_SPEC_FILTER**                   | The Agent SDK specific filter format for filtering out specific Kong services                                                                                                                                                                      |
| **KONG_SPEC_LOCALPATH**                | The local path that the agent will look in for API definitions                                                                                                                                                                                     |
| **KONG_SPEC_URLPATHS**                 | The URL paths that the agent will query on the gateway service for API definitions                                                                                                                                                                 |
| **KONG_SPEC_DEVPORTALENABLED**         | Set to true if the agent should look for spec files in the Kong Dev Portal (default: `false`)                                                                                                                                                      |
|                                        |                                                                                                                                                                                                                                                    |
| Traceability Agent Variables           |                                                                                                                                                                                                                                                    |
| **KONG_LOGS_HTTP_PATH**                | The path endpoint that the Traceability agent will listen on (default: `/requestlogs`)                                                                                                                                                             |
| **KONG_LOGS_HTTP_PORT**                | The port that the Traceability agent HTTP server will listen on (default: `9000`)                                                                                                                                                                  |

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

---
**NOTE:**

Don't forget to update your Amplify Central Region specific variables, such as the `CENTRAL_URL` setting.

All CENTRAL_* variables listed on [docs.axway.com](https://docs.axway.com/bundle/amplify-central/page/docs/connect_manage_environ/connect_api_manager/agent-variables/index.html) may be used on the Kong Agent.

---

### Kong setup

---
**NOTE:**

The Discovery agent expects that the Kong Gateway utilizes the [ACL](https://docs.konghq.com/hub/kong-inc/acl/) plugin to control access to the various routes provided in the Kong Gateway. On startup the agent checks that this plugin is in use prior to performing any discovery. The agent then uses this plugin while provisioning access to routes in Kong. [Provisioning Process](#provisioning-process).

---

#### Kong admin API secured by Kong Gateway

See [Kong - Securing the Admin API](https://docs.konghq.com/gateway/latest/production/running-kong/secure-admin-api/)

After following the procedures above the Kong Admin API can be secured using any authentication method that Kong provides. In this section you will learn the authentication types that the Kong agents support. As well as how to retrieve the values needed for the Kong agents.

Once the Kong admin API is secured a gateway service for it must be added to Kong and then a route configured to access the gateway service. After adding those configurations the following authentication may be added to the route. Then create a consumer, in Kong, for the agent and add credentials for that consumer. Note these credentials for later.

- Basic authentication
- API Key authentication
- OAuth2 authentication (currently, Kong returns an Internal Server Error if securing the admin api with OAuth2. The plugin can be created in Kong, but further requests will not work when receiving the token. The Agent is also configured to (as of now) not work with OAuth2)

#### Specification discovery methods

In order to publish a specification file that properly represents the gateway service configured in Kong, discovery agent supports two types of specification discovery methods. The first is a local directory, to the Kong agent, that specification files are saved in. The other is a list of URL paths that the Kong agent will query to attempt to find the specification file/

##### Local specification path

The local specification discovery method is configured by providing a value for the `KONG_SPEC_LOCALPATH` variable. When set the Kong agent will look for a tag on each of the available gateway services that are prefixed by `spec_local_`. When that tag is set the value, after stripping the prefix, is used to find the specification file in directory configured by `KONG_SPEC_LOCALPATH`. When this configuration value is set no other specification discovery methods will be used.

Ex.

Files on disk

```shell
> ls -l /path/to/specfiles
petstore.json
my-service.yaml
```

Configuration for agent

```shell
KONG_SPEC_LOCALPATH=/path/to/specfiles
```

Configuration on my-service gateway service

```json
{
...
"tags": [
  "tag1",
  "tag2",
  "spec_local_my-service.yaml",
  "tag3"
]
...
}
```

##### Filtering gateway services

Some possible ways to use the filter for gateway services (all these are done with the env var `KONG_SPEC_FILTER`):

Ex1: "tag.Any() == \"spec_local_petstore.json\"" -> this will find all the services that have a tag as "spec_local_petstore.json"
Ex2: "tag.discover.Exists()" -> this will find all tags that are equal to "discover"
Note: while both ways can achieve the same functionality, the first one is preferred because it does not restrict you on character usages for Kong tags (note the dot in example 2)

Currently, functionalities such as tag.Any().Contains() are not implemented in the SDK and only fully equal values are taken into account

##### URL specification paths

The URL specification paths discovery method is configured by value(s) for the `KONG_SPEC_URLPATHS` variable, comma separated. When values are set here, and a local path is not set, The Kong agent will query each of these paths against the gateway service in order to find a specification file. Once a specification file is found none of the other configured URL paths will be queried as that specification file will be used in the creation of the API Service on Central.

Ex.

Configuration for agent

```shell
KONG_SPEC_URLPATHS=/openapi.json,/swagger.json
```

##### Kong Dev Portal

The Kong Dev Portal discovery method is configured by providing a value for the `KONG_SPEC_DEVPORTALENABLED`, but also the local spec discovery needs to be disabled by setting an empty value for the`KONG_SPEC_LOCALPATH`, otherwise, the local discovery process will be used.

Ex.

Configuration for agent

```shell
KONG_SPEC_LOCALPATH=""
KONG_SPEC_DEVPORTALENABLED=true
```

#### HTTP Log plugin

The Traceability agent utilizes Kong's HTTP log plugin to track transactions. In order to set this up the plugin will have to be added, globally, and configured to send to the endpoint that the Traceability agent will listen on

- Navigate to the Plugins page
- Click *+ New Plugin*
- In the list of plugins find *HTTP Log* and click *enable*
- Ensure *Global* is selected so the agent receives events for all traffic
- Enter the following, all can be customized as necessary for your infrastructure, [HTTP Log](https://docs.konghq.com/hub/kong-inc/http-log/configuration/)
  - An Instance Name (optional)
  - Tags (optional)
  - content_type - `applicaiton/json`
  - custom_fields_by_lua - empty
  - flush_timeout - empty
  - headers - empty
  - http_endpoint - the endpoint the agent will listen on (ie. `http://traceability.host:9000/requestlogs`)
  - keepalive - `60000`
  - method - `POST`
  - queue.initial_retry_delay - `0.01`
  - queue.max_batch_size - `1000`
  - queue.max_bytes - empty
  - queue.max_coalescing_delay - `10`
  - queue.max_entries - `100000`
  - queue.max_retry_delay - `60`
  - queue.max_retry_time - `60`
  - queue_size - empty
  - retry_count - empty
  - timeout - `10000`
- Click *Install*

Kong is now setup to send transactions to the traceability agent.

## Kong agents deployment

The Kong agents are delivered as containers, kong_discovery_agent and kong_traceability_agent. These containers can be deployed directly to a container server, such as Docker, or using the provided helm chart. In this section you will lean how to deploy the agents directly as containers or within a kubernetes cluster using the helm chart.

### Additional information

Before beginning to deploy the agents following information will need to be gathered in addition to the details that were noted in setup.

- The full URL to connect to the Kong admin API, `KONG_ADMIN_URL`. Note that if secured by kong, the URL should look like: [https://host:port/secured-route-from-kong]
- The host the agent will use when setting the endpoint of a discovered API, (`KONG_PROXY_HOST`)
  - The HTTP `KONG_PROXY_PORTS_HTTP` and HTTPs `KONG_PROXY_PORTS_HTTPS` ports the agent will use with the endpoint above
- The URL paths, hosted by the gateway service, to query for spec files, `KONG_SPEC_URLPATHS`

### Docker

#### Environment variable files

In this section we will use the information gathered within the setup and additional information sections above and create two environment variable files for each agent to use. This is the minimum configuration assuming defaults for all other available settings. Note the settings below expect the use of the API Key authentication method for the [Kong admin api](#kong-admin-api-secured-by-kong-gateway).

Discovery Agent

```shell
KONG_ADMIN_URL=https://kong.url.com:8444
KONG_ADMIN_AUTH_APIKEY_HEADER="apikey"
KONG_ADMIN_AUTH_APIKEY_VALUE=123456789abcdefghijkl098765432109
KONG_PROXY_HOST=kong.proxy.endpoint.com
KONG_PROXY_PORTS_HTTP_VALUE=8000
KONG_PROXY_PORTS_HTTPS_VALUE=8443
KONG_SPEC_LOCALPATH=/specs

CENTRAL_ORGANIZATIONID=123456789
CENTRAL_AUTH_CLIENTID=kong-agents_123456789-abcd-efgh-ijkl-098765432109
CENTRAL_ENVIRONMENT=kong
CENTRAL_GRPC_ENABLED=true

AGENTFEATURES_MARKETPLACEPROVISIONING=true
```

Traceability Agent

```shell
CENTRAL_ORGANIZATIONID=123456789
CENTRAL_AUTH_CLIENTID=kong-agents_123456789-abcd-efgh-ijkl-098765432109
CENTRAL_ENVIRONMENT=kong
CENTRAL_GRPC_ENABLED=true

AGENTFEATURES_MARKETPLACEPROVISIONING=true
```

#### Deployment

In the following docker commands...

- `/home/user/keys` in the commands below refers to the directory where the key files were created during the last step in [Platform - service account](#platform---service-account)
- `/home/user/discovery/data:/data` and `/home/user/traceability/data:/data` are volumes that are used to store cached information to be saved outside of the container in order to persist restarts
- `/home/user/specs:/specs` is a volume mount for the spec files, the path in the `KONG_SPEC_LOCALPATH` variable is `/specs` and the path outside fo the container is `/home/user/specs`.
- `discovery-agents.env` and `traceability-agents.env` are files with the various environment variable settings that are available to each agent

Kong Discovery agent

```shell
docker run -d -v /home/user/keys:/keys -v /home/user/specs:/specs -v /home/user/discovery/data:/data --env-file discovery-agents.env ghcr.io/axway/kong_discovery_agent:latest
```

Kong Traceability agent

```shell
docker run -d -v /home/user/keys:/keys -v /home/user/traceability/data:/data --env-file traceability-agents.env -p 9000:9000 ghcr.io/axway/kong_traceability_agent:latest
```

### Helm

#### Traceability agent stateful set

The helm deployment of the Traceability agent uses a resource type of Stateful set along with a service to distribute the events to the agent pods. This is to allow scaling of the traceability agent in order to properly handle the load of events being sent through Kong. The agent is expected to be ran in the same kubernetes cluster as the Gateway and the [HTTP Log plugin](#http-log-plugin) should set its endpoint configuration to the [Service](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#services) that is created (ie.`http://kong-traceability-agent.kong-agents.svc.cluster.local:9000` where `kong-traceability-agent` is the service name and `kong-agents` is the namespace for the service)

#### Create secrets

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

#### Create volume, local specification files only

A volume of with the local specification files is required, given that is the desired [specification discovery method](#specification-discovery-methods). This volume could be of any kubernetes resource type which can be mounted in the Kong agent container. See [Kubernetes Volumes](https://kubernetes.io/docs/concepts/storage/volumes/).

Below are a couple of examples on adding specifications to a volume, of any type, to the agent pod for discovery purposes.

##### ConfigMap

Here is a sample of a ConfigMap that is used for the local specification files.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-spec-files
data:
  petstore.json: |
  ...spec file contents...
```

If a ConfigMap is being used, the kubectl command provides a utility to create the resource file for you. The command that follows will create a ConfigMap named `specs`, in the current kubernetes context and namespace. All files found in the current directories `specs/` folder will be included in the ConfigMap resource.

```bash
kubectl create configmap specs --from-file=specs/
```

---
**NOTE:**

An update to the ConfigMap will *NOT* be seen by any running pods, a pod restart would be required to see changes.

It is recommended to use a volume type that is more mutable than a ConfigMap. The agent has no knowledge of the volume type being used.

---

Once a resource with the files is created, which ever resource type is chosen, the overrides file will need to be updated with that resource information for mounting as a volume.

```yaml
kong:
  ...
  spec:
    localPath:
      configMap:             # type of the resource, provided in the deployment as a volume.
        name: my-spec-files  # name of the resource created
```

##### AWS S3 Synchronization

A kubernetes PersistentVolume resource with a CronJob volume can be set up to regularly synchronize spec files from an S3 bucket to the volume for the agent to utilize. Below you will find the three kubernetes resources that would need to be created as well as the update to the agnet helm chart override file.

- Create a PersistentVolume - this will store the specification files in the cluster
  - In this example a storage class of manual is used with a host path in the kubernetes cluster, however any class type may be used
    - [K8S Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
    - [EKS Persistent Volumes](https://aws.amazon.com/blogs/storage/persistent-storage-for-kubernetes/)

```yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: spec-volume
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/data"
```

- Create a PersistentVolumeClaim - this allows pods to mount this volume, needed for the job and the agent

```yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: spec-volume-claim
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

- Create a CronJob - this will run on the specified interval synchronizing the S3 bucket to the volume
  - The keys are embedded in this definition, but this can be replaced by a kubernetes secret or service account with the proper role in EKS
  - The schedule is to sync the spec files every 15 minutes
  - The bucket name is within the command, `specs-bucket`

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: s3-sync
spec:
  schedule: "*/15 * * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: s3-sync
            image: public.ecr.aws/aws-cli/aws-cli
            env:
            - name: AWS_ACCESS_KEY_ID
              value: XXXXXXXXXXXXXXXXXXX
            - name: AWS_SECRET_ACCESS_KEY
              value: XXXXXXXXXXXXXXXXXXX
            imagePullPolicy: IfNotPresent
            command:
            - /bin/sh
            - -c
            - aws s3 sync s3://specs-bucket/ /specs/
            volumeMounts:
            - name: specs-mount
              mountPath: /specs
          volumes:
          - name: specs-mount
            persistentVolumeClaim:
              claimName: spec-volume-claim
          restartPolicy: Never
```

- Override the agent helm chart accordingly
  
```yaml
kong:
  ...
  spec:
    localPath:
      persistentVolumeClaim:           # type of the resource, provided in the deployment as a volume.
        claimName: spec-volume-claim   # name of the resource created
```

#### Create overrides

overrides.yaml

```yaml
kong:
  enable:
    traceability: true # set this to true to deploy the traceability agent stateful set
  admin:
    url: http://kong-gateway-kong-admin.kong.svc.cluster.local:8001
  proxy:
    host: kong.proxy.endpoint.com
    ports:
      http: 80
      https: 443
  spec:
    localPath:
      configMap:            
        name: my-spec-files 
env:
  CENTRAL_ORGANIZATIONID: 123456789
  CENTRAL_AUTH_CLIENTID: kong-agents_123456789-abcd-efgh-ijkl-098765432109
  CENTRAL_ENVIRONMENT: kong
  CENTRAL_GRPC_ENABLED: true
  AGENTFEATURES_MARKETPLACEPROVISIONING: true
```

#### Deploy helm chart

Assuming you are already in the desired kubernetes context and namespace, execute the following commands.

Create the secret containing the Central key files used for authentication.

```shell
kubectl apply -f kong-agent-keys.yaml
```

Install the helm chart using the created overrides file. Set the release version to install.

```shell
release=v0.0.2
helm upgrade -i kong-agents https://github.com/Axway/agents-kong/releases/download/${release}/kong-agents.tgz -f overrides.yaml
```
