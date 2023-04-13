import fs from 'fs';
import { ethers } from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {Deployments, ChainConfig, chainType} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = new Deployments(new Map());

async function deploy_java(target: string, chain: any) {
  const iconNetwork = IconNetwork.getNetwork(target);
  console.log(`${target}: deploy BMC for ${chain.network}`)

  const bmcJar = JAVASCORE_PATH + '/bmc/build/libs/bmc-0.1.0-optimized.jar'
  const content = fs.readFileSync(bmcJar).toString('hex')
  const bmc = new Contract(iconNetwork)
  const deployTxHash = await bmc.deploy({
    content: content,
    params: {
      _net: chain.network
    }
  })
  const result = await bmc.getTxResult(deployTxHash)
  if (result.status != 1) {
    throw new Error(`BMC deployment failed: ${result.txHash}`);
  }
  console.log(`${target}: BMC deployed to ${bmc.address}`);

  deployments.set(target, {
    'network': chain.network,
    'contracts': {
      'bmc': bmc.address
    }
  })
}

async function deploy_solidity(target: string, chain: any) {
  console.log(`${target}: deploy BMC modules for ${chain.network}`)

  const BMCManagement = await ethers.getContractFactory("BMCManagement");
  const bmcm = await BMCManagement.deploy();
  await bmcm.deployed();
  await bmcm.initialize()
  console.log(`BMCManagement: deployed to ${bmcm.address}`);

  const BMCService = await ethers.getContractFactory("BMCService");
  const bmcs = await BMCService.deploy();
  await bmcs.deployed();
  await bmcs.initialize(bmcm.address)
  console.log(`BMCService: deployed to ${bmcs.address}`);

  const BMCPeriphery = await ethers.getContractFactory("BMCPeriphery");
  const bmcp = await BMCPeriphery.deploy();
  await bmcp.deployed();
  await bmcp.initialize(chain.network, bmcm.address, bmcs.address);
  console.log(`BMCPeriphery: deployed to ${bmcp.address}`);

  console.log(`${target}: management.setBMCPeriphery`);
  await bmcm.setBMCPeriphery(bmcp.address)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`${target}: management.setBMCService`);
  await bmcm.setBMCService(bmcs.address)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`${target}: service.setBMCPeriphery`);
  await bmcs.setBMCPeriphery(bmcp.address)
    .then((tx) => {
      return tx.wait(1)
    });

  deployments.set(target, {
    'network': chain.network,
    'contracts': {
      'bmcm': bmcm.address,
      'bmcs': bmcs.address,
      'bmc': bmcp.address,
    }
  })
}

async function main() {
  const link = ChainConfig.getLink();
  const srcChain: any = ChainConfig.getChain(link.src);
  const dstChain: any = ChainConfig.getChain(link.dst);
  console.log('link:', link)
  console.log('src:', srcChain)
  console.log('dst:', dstChain)

  await deploy_solidity(link.src, srcChain);
  await deploy_java(link.dst, dstChain);

  deployments.set('link', {
    'src': link.src,
    'dst': link.dst
  })
  deployments.save();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
