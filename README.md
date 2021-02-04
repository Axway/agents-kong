# Getting started

To download the package dependencies run `make download`

## Create an environment in Central

Log into Amplify Central https://apicentral.axway.com
Navigate to the Topology page
Click the "Environment" button in the top right.
Select "Other" for the gateway type.
Provide a name and a title, such as "kong-gateway" and then hit "Save" in the top right.

## Create a DOSA Account

Create a public and private key pair locally on your computer.
In Central, click the "Access" tab on the sidebar, which is the second to last tab.
Click on "Service Accounts".
Click the button in the top right that says "+ Service Account".
Name the account and provide the public key.

## Find your Organization ID

After making the environment click on your name in the top right. Select "Organization" from the dropdown.
You will see a field called "Organization ID". This will be needed to connect the agents to your org.

## Create a Kong user

Log into the kong manager. You will need to have a trial enterprise account. Ex: https://manager-radixlink2fbc76.kong-cloud.com/login
Click the Teams tab in the top navigation
Click the RBAC Users tab
Click the "Add New User" button
Provide a name for the user, and a value to use as a token, ex: 1234.
Click the Add/Edit Roles and add the "super-admin" role.

## Fill out the environment variables

Copy the content of `default_kong_discovery_agent.yml` to a new file named `kong_discovery_agent.yml`
Copy the content of `default_kong_traceability_agent.yml` to a new file named `kong_traceability_agent.yml`

In each of the two config files for the agents provide the following variables for your config.
Provide the `environment`, `organizationID`, `platformURL`, `team`, `url`, `clientID`, `privateKey`, `publicKey` (provide the full file path to the keys).

In the `kong_discovery_agent.yml` file provide the details of the kong user. `adminEndpoint`, `proxyEndpoint`, `proxyEndpointProtocols`, `user`, `token`

# Run the agents

## Development

Each agent is built and run independently

In development you can run an agent by running `go run ./cmd/discovery/discovery.go` or `go run ./cmd/discovery/traceability.go`. You do not need to build the binary for agents on every change.

## Build and run the binary

To build the discovery agent run `make build-disc`

To build the traceability agent run `make build-trace`

To run the discovery agent run `make run-disc`

To run the traceability agent run `make run-trace`

# Kong Discovery Agent

The discovery agent

The discovery agent has two mode to discover specs. Specs can be discovered from either the Kong Developer Portal by setting `specDevPortalEnabled` to `true`, or they can be discovered from a local directory.
To discover specs from a local directory provide a file path for the agent to look find specs in by setting the `specHomePath` field.

# Kong Traceability Agent