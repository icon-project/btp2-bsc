# End-to-End Testing Demo

In this demo, you will learn how to perform end-to-end testing between ICON and BSC (EVM-compatible) environment.

> **Note**
> The code in this folder is written specifically for the learning experience and is intended only for demonstration purposes.
> This demo code is not intended to be used in production.

## Prerequisites

To run the demo, the following software needs to be installed.

* Node.js 18 (LTS) \[[download](https://nodejs.org/en/download/)\]
* Docker compose (V2) \[[download](https://docs.docker.com/compose/install/)\]
* OpenJDK 11 or above \[[download](https://adoptium.net/)\]
* jq \[[download](https://github.com/stedolan/jq)\]
* go \[[download](https://go.dev/doc/install)\]


## Install required packages

This is a one-time setup procedure.
Before moving to the next step, you need to install all required packages via `npm` command.

```
npm install
```

A new directory named `node_modules` is created in the current working directory.

## Operate ICON/BSC Private Networks

To operate private networks where contracts for e2edemo can be installed, run the following command.

```
make start-nodes
```

## Build and Deploy contracts

Now it's time to build and deploy BTP and DApp contracts on the nodes.

To build all contracts, run the following command.

```
make build-all
```

It compiles both Java and Solidity contracts and generates artifacts for later deployment.

If no build errors were found, then you can deploy all the contracts using the following command.

```
make deploy-all
```

All contracts (BMC, BMV, xCall and DApp) have now been successfully deployed and linked on both the ICON and BSC chains.
The generated file, `deployments.json`, contains information needed to interact with the relays,
such as contract addresses and the network address of each chain.

The next step is to run a demo scenario script to verify the message delivery between two chains.

## Run Demo scenarios

Before running the demo scenario script, you need to spin up relay server to forward BTP messages to each chain.
Multiple terminal windows are required to complete these next steps.

Open a terminal window and run the following command to start the relay.

```
./relay.sh
```

You can now run the demo scenario script via the following command.

```
make run-demo
```

## Appendix

### Directory layout
| Directory                       | Description                                      |
|:--------------------------------|:-------------------------------------------------|
| docker                          | Docker-related files for ICON and BSC chains |
| scripts                         | Scripts for setup, deployment and demo scenarios |
| solidity                        | Root directory for compiling Solidity contracts  |
| solidity/contracts/dapp-sample  | Solidity contract for DApp sample                |
