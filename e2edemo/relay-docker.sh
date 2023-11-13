#!/bin/bash

docker run -d --network host -v $(pwd)/relay-config.json:/etc/config/config.json \
    -v $(pwd)/docker/bsc2/config/keystore.json:/keystore/bsc2/keystore.json \
    -v $(pwd)/docker/icon/config/keystore.json:/keystore/icon/keystore.json \
    -e RELAY_CONFIG=/etc/config/config.json \
    btp2-bsc/relay

