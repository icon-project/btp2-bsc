import fs from 'fs';
import { ethers } from 'hardhat';
import {Contract} from "../icon/contract";
import {BMC, BMV} from "../icon/btp";
import {Gov} from "../icon/system";
import {IconNetwork} from "../icon/network";
import IconService from "icon-sdk-js";
import {Deployments} from "./config";
const {IconConverter} = IconService;
const {JAVASCORE_PATH, BMV_BRIDGE} = process.env

const bridgeMode = BMV_BRIDGE == "true";
const deployments = Deployments.getDefault();
const iconNetwork = IconNetwork.getDefault();

let netTypeId = '';
let netId = '';

async function open_btp_network() {
  // open BTP network first before deploying BMV
  const icon = deployments.get('icon')
  const lastBlock = await iconNetwork.getLastBlock();
  const netName = `hardhat-${lastBlock.height}`
  console.log(`ICON: open BTP network for ${netName}`)
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
      netTypeId = indexed[1];
      netId = indexed[2];
    })
  console.log(`networkTypeId=${netTypeId}`);
  console.log(`networkId=${netId}`);
  icon.networkId = netId;
}

async function deploy_bmv() {
  // get last block number of ICON
  const lastBlock = await iconNetwork.getLastBlock();
  const icon = deployments.get('icon')
  icon.blockNum = lastBlock.height
  console.log(`Block number (${icon.network}): ${icon.blockNum}`);

  // get last block number of hardhat
  const blockNum = await ethers.provider.getBlockNumber();
  const hardhat = deployments.get('hardhat')
  hardhat.blockNum = blockNum
  console.log(`Block number (${hardhat.network}): ${hardhat.blockNum}`);

  // deploy BMV-Bridge java module
  const bmvJar = JAVASCORE_PATH + '/bmv/bsc/build/libs/bmv-bsc-0.1.0-optimized.jar'
  const content = fs.readFileSync(bmvJar).toString('hex')
  const bmv = new Contract(iconNetwork)
  const deployTxHash = await bmv.deploy({
    content: content,
    params: {
      chainId: '0x61',
      epoch: '0xc8',
      header: 'f90313a0bceed7be8740574304c2bcca6be1d856d47736116bb64620e907976800d304e4a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d4934794f474cf03cceff28abc65c9cbae594f725c80e12da0b3ba17d2e885c49963e9ecdbbe9da41d75c72efd429cac23d766324310f79e35a0ebd02912608adbc1a769803bd9ef4dd1137950d711db1cd4c9bd63304fa26cfaa031ca6b73ec082b1ae74e9c672bf290b5b310935ca53700580e2a6d09978f2ef6b9010000000000000000000000000000000000000000000000000000000000000000000200000000000000800000000000000000000000000000000004000000200000000000000000200000000008000000002010000000000000000201000000000020000020000200000000000000000000080000000000800000000010000000000001000000001000000000000000000000000400000000000000000000000820020000000000000000000000000000000000000800000000000000000000000000100002000200000000000100000000000000000000000000000002000000000010000000000008010000008000100000000000000000000000000000000008028401a9e5b08402faed858303395a84640ab1e4b90115d883010113846765746888676f312e31392e35856c696e757800000059afa6a31284214b9b9c85549ab3d2b972df0deef66ac2c935552c16704d214347f29fa77f77da6d75d7c7526d6247501b822fd4eaa76fcb64baea360279497f96c5d20b2a975c050e4220be276ace4892f4b41a980a75ecd1309ea12fa2ed87a8744fbfc9b863d5a2959d3f95eae5dc7d70144ce1b73b403b7eb6e0b71b214cb885500844365e95cd9942c7276e7fd8c5d15ef572d27f4f3087e17ee9099b760b1151c2f474cf03cceff28abc65c9cbae594f725c80e12d74f87b41bcf421a43d13612d5b54e87d118b70a74bcb3e7513fa38c6c88c09e52887cc13b633f6f52cae2bfb80db17cbd9252bc65d9e81b2cf1b97fb4ed4382101a00000000000000000000000000000000000000000000000000000000000000000880000000000000000',
      recents: [
          '980A75eCd1309eA12fa2ED87A8744fBfc9b863D5',
          'A2959D3F95eAe5dC7D70144Ce1b73b403b7EB6E0',
          'B71b214Cb885500844365E95CD9942C7276E7fD8',
          'C5D15eF572d27f4f3087e17Ee9099B760b1151c2',
          'F474Cf03ccEfF28aBc65C9cbaE594F725c80e12d'
      ],
      validators: [
          '1284214b9b9c85549aB3D2b972df0dEEf66aC2c9',
          '35552c16704d214347f29Fa77f77DA6d75d7C752',
          '6d6247501b822FD4Eaa76FCB64bAEa360279497f',
          '96C5D20b2a975c050e4220BE276ACe4892f4b41A',
          '980A75eCd1309eA12fa2ED87A8744fBfc9b863D5',
          'A2959D3F95eAe5dC7D70144Ce1b73b403b7EB6E0',
          'B71b214Cb885500844365E95CD9942C7276E7fD8',
          'C5D15eF572d27f4f3087e17Ee9099B760b1151c2',
          'F474Cf03ccEfF28aBc65C9cbaE594F725c80e12d'
      ]
      // _bmc: icon.contracts.bmc,
      // _net: hardhat.network,
      // _offset: IconConverter.toHex(hardhat.blockNum)
    }
  })
  const result = await bmv.getTxResult(deployTxHash)
  if (result.status != 1) {
      console.log(result);
    throw new Error(`BMV deployment failed: ${result.txHash}`);
  }
  icon.contracts.bmv = bmv.address
  console.log(`ICON BMV-Bridge: deployed to ${bmv.address}`);

  if (bridgeMode) {
    // deploy BMV-Bridge solidity module
    const BMVBridge = await ethers.getContractFactory("BMV")
    const bmvb = await BMVBridge.deploy(hardhat.contracts.bmcp, icon.network, icon.blockNum)
    await bmvb.deployed()
    hardhat.contracts.bmvb = bmvb.address
    console.log(`Hardhat BMV-Bridge: deployed to ${bmvb.address}`);
  } else {
    // get firstBlockHeader via btp2 API
    const networkInfo = await iconNetwork.getBTPNetworkInfo(netId);
    console.log('networkInfo:', networkInfo);
    console.log('startHeight:', '0x' + networkInfo.startHeight.toString(16));
    const receiptHeight = '0x' + networkInfo.startHeight.plus(1).toString(16);
    console.log('receiptHeight:', receiptHeight);
    const header = await iconNetwork.getBTPHeader(netId, receiptHeight);
    const firstBlockHeader = '0x' + Buffer.from(header, 'base64').toString('hex');
    console.log('firstBlockHeader:', firstBlockHeader);

    // deploy BMV-BtpBlock solidity module
    const BMVBtp = await ethers.getContractFactory("BtpMessageVerifier")
    const bmvBtp = await BMVBtp.deploy(hardhat.contracts.bmcp, icon.network, netTypeId, firstBlockHeader, '0x0')
    await bmvBtp.deployed()
    hardhat.contracts.bmv = bmvBtp.address
    console.log(`Hardhat BMV: deployed to ${bmvBtp.address}`);
  }

  // update deployments
  deployments.set('icon', icon)
  deployments.set('hardhat', hardhat)
  deployments.save();
}

async function setup_bmv() {
  const icon = deployments.get('icon')
  const hardhat = deployments.get('hardhat')

  // get the BTP address of ICON BMC
  const bmc = new BMC(iconNetwork, icon.contracts.bmc)
  const bmv = new BMV(iconNetwork, icon.contracts.bmv)
  const bmcIconAddr = await bmc.getBtpAddress()
  console.log(`BTP address of ICON BMC: ${bmcIconAddr}`)

  // get the BTP address of hardhat BMC
  const bmcm = await ethers.getContractAt('BMCManagement', hardhat.contracts.bmcm)
  const bmcp = await ethers.getContractAt('BMCPeriphery', hardhat.contracts.bmcp)
  const bmcHardhatAddr = await bmcp.getBtpAddress()
  console.log(`BTP address of Hardhat BMC: ${bmcHardhatAddr}`)

  console.log(`ICON: addVerifier for ${hardhat.network}`)
  await bmc.addVerifier(hardhat.network, bmv.address)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to register BMV to BMC: ${result.txHash}`);
      }
    })
  console.log(`ICON: addBTPLink for ${bmcHardhatAddr}`)
  await bmc.addBTPLink(bmcHardhatAddr, netId)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addBTPLink: ${result.txHash}`);
      }
    })
  console.log(`ICON: addRelay`)
  await bmc.addRelay(bmcHardhatAddr, iconNetwork.wallet.getAddress())
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addRelay: ${result.txHash}`);
      }
    })

  console.log(`Hardhat: addVerifier for ${icon.network}`)
  let bmvAddress;
  if (bridgeMode) {
    const bmvb = await ethers.getContractAt('BMV', hardhat.contracts.bmvb)
    bmvAddress = bmvb.address;
  } else {
    const bmvBtp = await ethers.getContractAt('BtpMessageVerifier', hardhat.contracts.bmv)
    bmvAddress = bmvBtp.address;
  }
  await bmcm.addVerifier(icon.network, bmvAddress)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`Hardhat: addLink: ${bmcIconAddr}`)
  await bmcm.addLink(bmcIconAddr)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`Hardhat: addRelay`)
  const signers = await ethers.getSigners()
  await bmcm.addRelay(bmcIconAddr, signers[0].getAddress())
    .then((tx) => {
      return tx.wait(1)
    });
}

open_btp_network()
  .then(deploy_bmv)
  .then(setup_bmv)
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
