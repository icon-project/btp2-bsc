#!/usr/bin/env bash

workspace=$(cd `dirname $0`; pwd)/..
#source ${workspace}/.env

function prepare() {
   if ! [[ -f /usr/local/bin/geth ]];then
        echo "geth do not exist!"
        exit 1
   fi
   rm -rf ${workspace}/storage/*
   cd ${workspace}/genesis
   rm -rf validators.conf
}

function init_validator() {
    name=$1
    echo "[INIT_VALIDATOR] name: ${name}"
    mkdir -p ${workspace}/storage/${name}
    echo "${KEYPASS}" > ${workspace}/storage/${name}/passwd.txt
    consensus_address=$(geth account new --datadir ${workspace}/storage/${name} --password ${workspace}/storage/${name}/passwd.txt | grep "Public address of the key:" | awk -F"   " '{print $2}')

    echo "consensus address: ${consensus_address}"
    expect /root/scripts/create_bls_key.sh ${workspace}/storage/${name}
    echo "${BLS_WAL_PW}" > ${workspace}/storage/${name}/bls/passwd.txt
    vote_address=0x$(cat ${workspace}/storage/${name}/bls/keystore/*json | jq .pubkey | sed 's/"//g')
    echo "vote address: ${vote_address}"

    powers="0x000001d1a94a2000"
    echo "${consensus_address},${consensus_address},${consensus_address},${powers},${vote_address}" >> ${workspace}/genesis/validators.conf
}

function generate_genesis() {
    echo "[GENERATE_GENESIS]"
    holder=0x$(cat ${workspace}/config/holder.json | jq .address | sed 's/"//g')
    echo "INITIAL HOLDER: $holder"
    sed "s/{{INIT_HOLDER_ADDR}}/${holder}/g" ${workspace}/config/init_holders.template > ${workspace}/genesis/init_holders.js
    sed -i -e "s/numOperator = 2;/operators[VALIDATOR_CONTRACT_ADDR] = true;\noperators[SLASH_CONTRACT_ADDR] = true;\nnumOperator = 4;/g" ${workspace}/genesis/contracts/SystemReward.template
    sed -i -e "s/for (uint i; i<validatorSetPkg.validatorSet.length; ++i) {/ValidatorExtra memory validatorExtra;\nfor (uint i; i<validatorSetPkg.validatorSet.length; ++i) {\n validatorExtraSet.push(validatorExtra);\n validatorExtraSet[i].voteAddress=validatorSetPkg.voteAddrs[i];/g" ${workspace}/genesis/contracts/BSCValidatorSet.template
    sed -i -e "s/\"0x\" + publicKey.pop()/vs[4]/g" ${workspace}/genesis/generate-validator.js

    node generate-validator.js
    node generate-genesis.js --chainid ${BSC_CHAIN_ID} --bscChainId "$(printf '%04x\n' ${BSC_CHAIN_ID})"
}

function init_genesis_data() {
    echo "[INIT_GENESIS_DATA]"
    name=$1
    nth=$2

    geth --datadir ${workspace}/storage/${name} init ${workspace}/genesis/genesis.json
    cp ${workspace}/config/config.toml ${workspace}/storage/${name}/config.toml
    sed -i -e "s/{{NetworkId}}/${BSC_CHAIN_ID}/g" ${workspace}/storage/${name}/config.toml
}

function update_static_peers() {
    num=$1
    target=$2
    staticPeers=""
    for ((j=1;j<=$num;j++)); do
        if [ $j -eq $target ]
        then
           continue
        fi

        file=${workspace}/storage/node${j}/geth/nodekey
        port=30311
        domain=node${j}
        if [ ! -z "$staticPeers" ]
        then
            staticPeers+=","
        fi
        staticPeers+='"'"enode:\/\/$(bootnode -nodekey $file -writeaddress)@$domain:$port"'"'
    done

    static_nodes=$(echo "${staticPeers}")
    sed -i -e "s/{{StaticNodes}}/${static_nodes}/g" ${workspace}/storage/node${target}/config.toml
}

prepare
for((i=1;i<=${NUMS_OF_VALIDATOR};i++)); do
     init_validator "node${i}"
done
generate_genesis
for((i=1;i<=${NUMS_OF_VALIDATOR};i++)); do
     init_genesis_data "node${i}" ${i}
done
for((i=1;i<=${NUMS_OF_VALIDATOR};i++)); do
    update_static_peers ${NUMS_OF_VALIDATOR} ${i}
done
echo "[COMPLETE] Network Bootstrap"
