import rlp from 'rlp';
import { ethers } from 'hardhat';
import {Contract, IconNetwork, Jar} from "../icon";
import {Gov, BMC, BMV, getBtpAddress} from "../icon";
import IconService from "icon-sdk-js";
import {Deployments, chainType, DEFAULT_CONFIRMATIONS} from "./config";
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

async function deploy_bmv_jav(srcNetwork: IconNetwork, srcChain: any, params: any) {
  const content = Jar.readFromFile(JAVASCORE_PATH, "bmv/bsc2");
  console.log('src network:', srcNetwork);
  console.log('params:', params);
  const bmv = new Contract(srcNetwork)
  const deployTxHash = await bmv.deploy({
    content: content,
    params,
  })
  const result = await bmv.getTxResult(deployTxHash);
  if (result.status != 1) {
    throw new Error(`BMV deployment failed: ${result.txHash}`);
  }
  srcChain.contracts.bmv = bmv.address;
  console.log(`${srcChain.network}: BMV-BTPBlock: deployed to ${bmv.address}`);
}

const ValidatorBytesLength = 20 *2;

async function headByNumber(number: number) {
    return await ethers.provider.send('eth_getBlockByNumber', ['0x' + number.toString(16), false]);
}

async function genJavBmvParams(bmc: string, number: number) {
  const curnum = number != undefined ? number : await ethers.provider.getBlockNumber();
  const tarnum = curnum - curnum % EPOCH;
  console.log('trusted block number:', tarnum);
  const curr = await headByNumber(tarnum);
  const prev = tarnum != 0 ? await headByNumber(tarnum - EPOCH) : curr;
  let validators = parseValidators(Buffer.from(prev.extraData.slice(2, prev.extraData.length), 'hex'));
  let candidates = parseValidators(Buffer.from(curr.extraData.slice(2, curr.extraData.length), 'hex'));

  console.log('validators:', validators);

  const recents = [];
  for (let i = Math.floor(validators.length / 2); i >= 0; i--) {
    let miner = (await ethers.provider.getBlock(tarnum-i)).miner;
      recents.push(miner);
  }

  return {
      _bmc: bmc,
      _chainId: '0x' + (await ethers.provider.getNetwork()).chainId.toString(16),
      _header: Buffer.from(rlp.encode([
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
        curr.nonce,
        curr.baseFeePerGas
      ])).toString('hex'),
      _recents: Buffer.from(rlp.encode(recents)).toString('hex'),
      _candidates: Buffer.from(rlp.encode(candidates)).toString('hex'),
      _validators: Buffer.from(rlp.encode(validators)).toString('hex')
  }
}
const BlsPubLenth = 48;
const EthAddrLength = 20;
const ValidatorBytesLengthLuban = EthAddrLength + BlsPubLenth;
const ExtraVanity = 32;
const ExtraSeal = 65;
const ValidatorNumberSize = 1;

function parseValidators(extra) {
  console.log('etra:', extra.toString('hex'));
  if (extra.length <= ExtraVanity + ExtraSeal) {
    throw new Error("Wrong Validator Bytes");
  }

  const num = Buffer.from(extra, 'hex')[ExtraVanity];
  const start = ExtraVanity + 1;
  const end = start + num * ValidatorBytesLengthLuban;
  const validatorsBytes = Buffer.from(extra.slice(start, end));
  let validators = [];
  for (let i = 0; i < num; i++) {
    validators.push([
      '0x' + validatorsBytes.slice(i * ValidatorBytesLengthLuban, i * ValidatorBytesLengthLuban + EthAddrLength).toString('hex'),
      '0x' + validatorsBytes.slice(i * ValidatorBytesLengthLuban + EthAddrLength, (i+1) * ValidatorBytesLengthLuban).toString('hex')
    ]);
  }
  return validators;
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

  console.log(`${src}: addVerifier for ${dstChain.network}, ${srcChain.contracts.bmv}`)
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
      return tx.wait(DEFAULT_CONFIRMATIONS)
    });
  console.log(`${src}: addLink: ${dstBmcAddr}`)
  await bmcm.addLink(dstBmcAddr)
    .then((tx) => {
      return tx.wait(DEFAULT_CONFIRMATIONS)
    });
  const signer = await (await ethers.getSigners())[0].getAddress()
  console.log(`${src}: addRelay: ${signer}`)
  await bmcm.addRelay(dstBmcAddr, signer)
    .then((tx) => {
      return tx.wait(DEFAULT_CONFIRMATIONS)
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
  let params = await genJavBmvParams(dstChain.contracts.bmc);
  console.log('bmv deployment args:', params);
  await deploy_bmv_sol(src, dst, srcChain, dstChain);
  console.log('try jav deploy');
  await deploy_bmv_jav(IconNetwork.getNetwork(dst), dstChain, params);
  console.log('done jav deploy');
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
