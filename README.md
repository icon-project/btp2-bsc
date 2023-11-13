# Relay System for BTP 2.0

## Introduction

This is a reference `relay` implementation for BTP 2.0 protocol.

### Target chains
* ICON (BTP Block)
* BSC

### Terminologies

| Word            | Description                                                                                                                             |
|:----------------|:----------------------------------------------------------------------------------------------------------------------------------------|
| BTP             | Blockchain Transmission Protocol, see [ICON BTP Standard](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md).             |
| BTP Message     | A verified message which is delivered by the relay                                                                                      |
| Service Message | A payload in a BTP Message                                                                                                              |
| Relay Message   | A message including BTP Messages with proofs for that, and other block update messages.                                                 |
| Network Address | A string to identify blockchain network <br/> *ex) 0x1.icon*                                                                            |
| BTP Address     | A string of URL for locating an account of the blockchain network <br/> *ex) btp://0x1.icon/cx87ed9048b594b95199f326fc76e76a9d33dd665b* |

See [ICON BTP Standard](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md) and [Relay System for BTP 2.0](https://github.com/icon-project/btp2) for more details.


### Components

* [BTP Message Center (BMC)](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md#btp-message-center) - smart contract
  - Receive BTP messages through transactions.
  - Send BTP messages through events.

* [BTP Message Verifier (BMV)](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md#btp-message-verifier) - smart contract
  - Update blockchain verification information
  - Verify delivered BTP message and decode it

* [BTP Service Handler (BSH)](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md#btp-service-handler) - smart contract
  - Handle service messages related to the service.
  - Send service messages through the BMC

* [BTP Message Relay (BMR)](doc/bmr.md) - external software (implemented by this repository)
  - Monitor BTP events
  - Gather proofs for the events
  - Send BTP Relay Message

## Getting Started
### Build Relay
```shell
`git clone https://github.com/icon-project/btp2-bsc.git --recurse-submodules`
`cd btp2-bsc`

# compile binary
`make relay`

# build docker image
make relay-image
```

### Configuration
For retrieving `chains_config.src.options.start_number` please read the [Retrieve start-block number](#Retrieve-start-block-number)
```json
{
  "relay_config": {
    "base_dir": ".",
    "direction": "both",
    "log_level": "debug",
    "console_level": "trace",
    "log_forwarder": {
      "vendor": "",
      "address": "",
      "level": "info",
      "name": ""
    },
    "log_writer": {
      "filename": "data/relay.log",
      "maxsize": 100,
      "maxage": 0,
      "maxbackups": 0,
      "localtime": false,
      "compress": false
    }
  },
  "chains_config": {
    "src": {
      "address": "btp://0x63.bsc/0x9Fd9e050682A8795dEa6eE70870A82a513d390Ac",
      "endpoint": "https://bsc-node",
      "key_store": "${BSC_KEYSTORE_FILE}",
      "key_password": "-",
      "options": {
        "db_type": "leveldb",
        "db_path": "data",
        "start_number": "32489600"
      },
      "type" : "bsc-hertz"
    },
    "dst": {
      "address": "btp://0x3.icon/cxf1b0808f09138fffdb890772315aeabb37072a8a",
      "endpoint": "https://icon-node/api/v3/icon_dex",
      "key_store": "${ICON_KEYSTORE_FILE}",
      "key_password": "-",
      "type" : "icon-btpblock"
    }
  }
}
```

### Execution
```shell
./bin/relay start -c config.json
```
or, execution with docker

```shell
docker run -d --network=host -v ${BSC_KEYSTORE_FILE}:${BSC_KEYSTORE_FILE_ON_DOCKER} \
  -v ${ICON_KEYSTORE_FILE}:${ICON_KEYSTORE_FILE_ON_DOCKER} \
  -v ${CONFIG_FILE}:/etc/config/config.json -e RELAY_CONFIG=/etc/config/config.json \
  -d btp2-bsc/relay
```

### Retrieve start block number

```shell
node e2edemo/scripts/setup/retrieve-start-block-number.js ${ICON-NODE-RPC-URL} ${BTP-ADDRESS-BMC-ON-BSC} ${BMC-ADDRESS-ON-ICON}

example:) node ./e2edemo/scripts/setup/retrieve-start-block-number.js http://localhost:9080/api/v3/icon_dex btp://0x63.bsc/0x61143C6026C0459847389fB5Cb8f5A6482A6b5D4 cxc762bf1b69625dc98e8a872b8b8dc3468eea28f0
```

or

```shell
1. Retrive BTP Status for BSC
curl --location 'https://icon-rpc-url' \
    --header 'Content-Type: application/json' \
    --data '{
      "jsonrpc": "2.0",
      "method": "icx_call",
      "id": 1,
      "params": {
          "to": "cxc762bf1b69625dc98e8a872b8b8dc3468eea28f0" // contract address of bmc on icon
          "dataType": "call",
          "data": {
              "method": "getStatus",
              "params": {
                  "_link": "btp://0x63.bsc/0x61143C6026C0459847389fB5Cb8f5A6482A6b5D4" // btp address of bmc on bsc
              }
          }
      }
    }'

// Response
{
    "jsonrpc":"2.0",
    "result":{
        "cur_height": "0xed02bb",
        "rx_seq": "0x288",
        "tx_seq": "0x362",
        "verifier": {
            "extra": "0xf86d8401efc080f86601a05af60449fa89cd17ddbd129939db8a46e51d6489a09e370e0849a71f6febabaf01a0fb97f327a8a677837d0eebe25e23d3eb9e9b6b8fbdd39abf42ad5308fd3a18c280a0a38c2f7e262db3b848d742c71647425ec3997dbcf4c319dec783d553ecc82040",
            "height": "0x211159b"
        }
    },
    "id":1
}

2. RLP(n)-Decode `extra` field of BTP Status
$decode(f86d8401efc080f86601a05af60449fa89cd17ddbd129939db8a46e51d6489a09e370e0849a71f6febabaf01a0fb97f327a8a677837d0eebe25e23d3eb9e9b6b8fbdd39abf42ad5308fd3a18c280a0a38c2f7e262db3b848d742c71647425ec3997dbcf4c319dec783d553ecc82040)

[
  # 1st element means start block number
  <Buffer 01 ef c0 80>, // decimals format: 32489600
  [
    <Buffer 01>,
    <Buffer 5a f6 04 49 fa 89 cd 17 dd bd 12 99 39 db 8a 46 e5 1d 64 89 a0 9e 37 0e 08 49 a7 1f 6f eb ab af>,
    <Buffer 01>,
    <Buffer fb 97 f3 27 a8 a6 77 83 7d 0e eb e2 5e 23 d3 eb 9e 9b 6b 8f bd d3 9a bf 42 ad 53 08 fd 3a 18 c2>,
    Uint8Array(0) [],
    <Buffer a3 8c 2f 7e 26 2d b3 b8 48 d7 42 c7 16 47 42 5e c3 99 7d bc f4 c3 19 de c7 83 d5 53 ec c8 20 40>
  ]
]

```

## E2E Testing Demo
* Follow the instruction in [End-to-End Testing Demo](e2edemo) folder.

## References
* [ICON BTP Standard](https://github.com/icon-project/IIPs/blob/master/IIPS/iip-25.md)
