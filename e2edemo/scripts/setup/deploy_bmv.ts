import fs from 'fs';
import rlp from 'rlp';
import { ethers } from 'hardhat';
import {Contract} from "../icon/contract";
import {BMC, BMV, getBtpAddress} from "../icon/btp";
import {Gov} from "../icon/system";
import {IconNetwork} from "../icon/network";
import IconService from "icon-sdk-js";
import {Deployments, chainType} from "./config";
const {IconConverter} = IconService;
const {JAVASCORE_PATH, BMV_BRIDGE} = process.env
const EPOCH = 200;

const deployments = Deployments.getDefault();

async function open_btp_network(src: string, dst: string, icon: any) {
  // open BTP network first before deploying BMV
  const iconNetwork = IconNetwork.getNetwork(src);
  const lastBlock = await iconNetwork.getLastBlock();
  const netName = `${dst}-${lastBlock.height}`
  console.log(`${src}: open BTP network for ${netName}`)
  const gov = new Gov(iconNetwork);
  await gov.openBTPNetwork(netName, icon.contracts.bmc)
    .then((txHash) => gov.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to openBTPNetwork: ${result.txHash}`);
      }
      return gov.filterEvent(result.eventLogs,
        'BTPNetworkOpened(int,int)', 'cx0000000000000000000000000000000000000000')
    })
    .then((events) => {
      console.log(events);
      if (events.length == 0) {
        throw new Error(`ICON: failed to find networkId`);
      }
      const indexed = events[0].indexed || [];
      const netTypeId = indexed[1];
      const netId = indexed[2];
      console.log(`${src}: networkTypeId=${netTypeId}`);
      console.log(`${src}: networkId=${netId}`);
      icon.networkTypeId = netTypeId;
      icon.networkId = netId;
    })
}

async function get_first_btpblock_header(network: IconNetwork, chain: any) {
  // get firstBlockHeader via btp2 API
  const networkInfo = await network.getBTPNetworkInfo(chain.networkId);
  console.log('networkInfo:', networkInfo);
  console.log('startHeight:', '0x' + networkInfo.startHeight.toString(16));
  const receiptHeight = '0x' + networkInfo.startHeight.plus(1).toString(16);
  console.log('receiptHeight:', receiptHeight);
  const header = await network.getBTPHeader(chain.networkId, receiptHeight);
  const firstBlockHeader = '0x' + Buffer.from(header, 'base64').toString('hex');
  console.log('firstBlockHeader:', firstBlockHeader);
  return firstBlockHeader;
}

async function deploy_bmv_jav(src: string, dst: string, srcChain: any, dstChain: any, params: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const bmvJar = JAVASCORE_PATH + '/bmv/bsc/build/libs/bmv-bsc-0.1.0-optimized.jar';
  const content = fs.readFileSync(bmvJar).toString('hex');
  const bmv = new Contract(srcNetwork)
  const deployTxHash = await bmv.deploy({
    content: content,
    params: params
  });
  const result = await bmv.getTxResult(deployTxHash);
  if (result.status != 1) {
    throw new Error(`BMV deployment failed: ${result.txHash}`);
  }
  srcChain.contracts.bmv = bmv.address;
  console.log(`${srcChain.network}: BMV-BTPBlock: deployed to ${bmv.address}`);
}

const ValidatorBytesLength = 20 *2;

async function headByNumber(number: number) {
    return await ethers.provider.send('eth_getHeaderByNumber', ['0x' + number.toString(16)]);
}

async function genJavBmvParams(bmc: string, number: number) {
  const curnum = number != undefined ? number : await ethers.provider.getBlockNumber();
  const tarnum = curnum - curnum % EPOCH;
  console.log('trusted block number:', tarnum);
  const curr = await headByNumber(tarnum);
  const prev = tarnum != 0 ? await headByNumber(tarnum - EPOCH) : curr;
  const extra = prev.extraData;
  const valBytes = extra.slice('0x'.length + 32 * 2, extra.length - 65 * 2)
  const validators = [];
  for (let i = 0; i < valBytes.length / (ValidatorBytesLength); i++) {
      validators.push('0x' + valBytes.slice(i*ValidatorBytesLength, (i+1)*ValidatorBytesLength));
  }

  const recents = [];
  for (let i = Math.floor(validators.length / 2); i >= 0; i--) {
      recents.push((await ethers.provider.getBlock(tarnum-i)).miner);
  }

  return {
      bmc: bmc,
      chainId: '0x' + (await ethers.provider.getNetwork()).chainId.toString(16),
      header: Buffer.from(rlp.encode([
        curr.parentHash,
        curr.sha3Uncles,
        curr.miner,
        curr.stateRoot,
        curr.transactionsRoot,
        curr.receiptsRoot,
        curr.logsBloom,
        curr.difficulty,
        curr.number,
        curr.gasLimit,
        curr.gasUsed,
        curr.timestamp,
        curr.extraData,
        curr.mixHash,
        curr.nonce
      ])).toString('hex'),
      recents: recents,
      validators: validators
  }
}

async function deploy_bmv_sol(src: string, dst: string, srcChain: any, dstChain: any) {
  const blockNum = await ethers.provider.getBlockNumber();
  srcChain.blockNum = blockNum
  console.log(`${src}: block number (${srcChain.network}): ${srcChain.blockNum}`);

  const dstNetwork = IconNetwork.getNetwork(dst);
  const lastBlock = await dstNetwork.getLastBlock();
  dstChain.blockNum = lastBlock.height
  console.log(`${dst}: block number (${dstChain.network}): ${dstChain.blockNum}`);

  // deploy BMV-BTPBlock solidity for dst network
  const firstBlockHeader = await get_first_btpblock_header(dstNetwork, dstChain);
  const BMVBtp = await ethers.getContractFactory("BtpMessageVerifier");
  const bmvBtp = await BMVBtp.deploy(srcChain.contracts.bmc, dstChain.network, dstChain.networkTypeId, firstBlockHeader, '0x0');
  await bmvBtp.deployed()
  srcChain.contracts.bmv = bmvBtp.address
  console.log(`${dst}: BMV: deployed to ${bmvBtp.address}`);
}

async function setup_link_icon(src: string, srcChain:any, dstChain: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const bmc = new BMC(srcNetwork, srcChain.contracts.bmc);
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);

  console.log(`${src}: addVerifier for ${dstChain.network}`)
  await bmc.addVerifier(dstChain.network, srcChain.contracts.bmv)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to register BMV to BMC: ${result.txHash}`);
      }
    })
  console.log(`${src}: addBTPLink for ${dstBmcAddr}`)
  await bmc.addBTPLink(dstBmcAddr, srcChain.networkId)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addBTPLink: ${result.txHash}`);
      }
    })
  console.log(`${src}: addRelay`)
  await bmc.addRelay(dstBmcAddr, srcNetwork.wallet.getAddress())
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addRelay: ${result.txHash}`);
      }
    })
}

async function setup_link_sol(src: string, srcChain: any, dstChain: any) {
  const bmcm = await ethers.getContractAt('BMCManagement', srcChain.contracts.bmcm)
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);

  console.log(`${src}: addVerifier for ${dstChain.network}`)
  await bmcm.addVerifier(dstChain.network, srcChain.contracts.bmv)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`${src}: addLink: ${dstBmcAddr}`)
  await bmcm.addLink(dstBmcAddr)
    .then((tx) => {
      return tx.wait(1)
    });
  const signer = await (await ethers.getSigners())[0].getAddress()
  console.log(`${src}: addRelay: ${signer}`)
  await bmcm.addRelay(dstBmcAddr, signer)
    .then((tx) => {
      return tx.wait(1)
    });
}

(async () => {
  const src = deployments.getSrc();
  const dst = deployments.getDst();
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);

  console.log('src:', src, 'dst:', dst);
  await open_btp_network(dst, src, dstChain);

  console.log('srcChain:', srcChain, 'dstChain:', dstChain);
  await deploy_bmv_sol(src, dst, srcChain, dstChain);
  await deploy_bmv_jav(dst, src, dstChain, srcChain, await genJavBmvParams(dstChain.contracts.bmc));
  // update deployments
  deployments.set(src, srcChain);
  deployments.set(dst, dstChain);
  deployments.save();

  await setup_link_icon(dst, dstChain, srcChain);
  await setup_link_sol(src, srcChain, dstChain);

})().catch((error) => {
    console.error(error);
    process.exitCode = 1
});
