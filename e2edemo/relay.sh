#!/bin/bash

RELAY_BIN=../bin/relay
DEPLOYMENTS=deployments.json
CHAIN_CONFIG=chain_config.json


DB_TYPE=leveldb
DB_PATH=./data
START_NUMBER=400

if [ ! -f ${RELAY_BIN} ]; then
    (cd ..; make relay)
fi

SRC=$(cat ${CHAIN_CONFIG} | jq -r .link.src)
DST=$(cat ${CHAIN_CONFIG} | jq -r .link.dst)

SRC_NETWORK=$(cat ${DEPLOYMENTS} | jq -r .${SRC}.network)
DST_NETWORK=$(cat ${DEPLOYMENTS} | jq -r .${DST}.network)
SRC_BMC_ADDRESS=$(cat ${DEPLOYMENTS} | jq -r .${SRC}.contracts.bmc)
DST_BMC_ADDRESS=$(cat ${DEPLOYMENTS} | jq -r .${DST}.contracts.bmc)

# SRC network config
SRC_ADDRESS=btp://${SRC_NETWORK}/${SRC_BMC_ADDRESS}
SRC_ENDPOINT=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.endpoint)
SRC_KEY_STORE=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.keystore)
SRC_KEY_PASSWORD=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.keypass)

# DST network config
DST_ADDRESS=btp://${DST_NETWORK}/${DST_BMC_ADDRESS}
DST_ENDPOINT=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.endpoint)
DST_KEY_STORE=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.keystore)
DST_KEY_PASSWORD=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.keypass)

SRC_NETWORK_NAME=$(echo ${SRC_NETWORK} | cut -d. -f2)
DST_NETWORK_NAME=$(echo ${DST_NETWORK} | cut -d. -f2)

# Determine src type
if [ $SRC_NETWORK_NAME == icon ]; then
  SRC_TYPE="icon-btpblock"
else
  SRC_TYPE="bsc-hertz"
fi

# Determine src type
if [ "x$BMV_BRIDGE" = xtrue ]; then
  echo "Using Bridge mode"
  DST_TYPE="icon-bridge"
else
  echo "Using BTPBlock mode"
  DST_TYPE="icon-btpblock"
fi

get_bsc_option() {
    echo '{
      "db_type": "'$1'",
      "db_path": "'$2'",
      "start_number": "'$3'"
    }' | tr -d [:space:]
}

BSC_OPTION=$(get_bsc_option "$DB_TYPE" "$DB_PATH" "$START_NUMBER")

get_config() {
  if [ $# -gt 5 -a ${#6} -gt 0 ]; then
    echo '{
      "address": "'$1'",
      "endpoint": "'$2'",
      "key_store": "'$3'",
      "key_password": "'$4'",
      "type": "'$5'",
      "options": '$6'
    }' | tr -d [:space:]
  else
    echo '{
      "address": "'$1'",
      "endpoint": "'$2'",
      "key_store": "'$3'",
      "key_password": "'$4'",
      "type": "'$5'"
    }' | tr -d [:space:]
  fi
}
SRC_CONFIG=$(get_config "$SRC_ADDRESS" "$SRC_ENDPOINT" "$SRC_KEY_STORE" "$SRC_KEY_PASSWORD" "$SRC_TYPE" "$BSC_OPTION")
DST_CONFIG=$(get_config "$DST_ADDRESS" "$DST_ENDPOINT" "$DST_KEY_STORE" "$DST_KEY_PASSWORD" "$DST_TYPE")

echo ${SRC_CONFIG}
echo ${DST_CONFIG}
${RELAY_BIN} \
    --base_dir .relay \
    --direction both \
    --src_config ${SRC_CONFIG} \
    --dst_config ${DST_CONFIG} \
    start