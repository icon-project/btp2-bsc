#!/usr/bin/env bash

DATA_DIR=/root/.ethereum

ADDRESS="0x$(cat ${DATA_DIR}/keystore/* | jq -r .address)"

echo "NODE ETH ADDRESS: $ADDRESS"

geth --config ${DATA_DIR}/config.toml --datadir ${DATA_DIR} --password ${DATA_DIR}/passwd.txt \
    --nodekey ${DATA_DIR}/geth/nodekey -unlock $ADDRESS --rpc.allow-unprotected-txs \
    --allow-insecure-unlock --gcmode archive --syncmode=full --mine --vote
