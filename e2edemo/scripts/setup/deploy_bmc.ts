import {ethers} from 'hardhat';
import {Contract, IconNetwork, Jar} from "../icon";
import {Deployments, ChainConfig, chainType, DEFAULT_CONFIRMATIONS } from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = new Deployments(new Map());

async function deploy_java(target: string, chain: any) {
  const iconNetwork = IconNetwork.getNetwork(target);
  console.log(`${target}: deploy BMC for ${chain.network}`)

  const content = Jar.readFromFile(JAVASCORE_PATH, "bmc");
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
  await bmcm.deployTransaction.wait(DEFAULT_CONFIRMATIONS);
  await (await bmcm.initialize()).wait(DEFAULT_CONFIRMATIONS);
  console.log(`BMCManagement: deployed to ${bmcm.address}`);

  const BMCService = await ethers.getContractFactory("BMCService");
  const bmcs = await BMCService.deploy();
  await bmcs.deployTransaction.wait(DEFAULT_CONFIRMATIONS);
  await (await bmcs.initialize(bmcm.address)).wait(DEFAULT_CONFIRMATIONS);
  console.log(`BMCService: deployed to ${bmcs.address}`);

  const BMCPeriphery = await ethers.getContractFactory("BMCPeriphery");
  const bmcp = await BMCPeriphery.deploy();
  await bmcp.deployTransaction.wait(DEFAULT_CONFIRMATIONS);
  await (await bmcp.initialize(chain.network, bmcm.address, bmcs.address)).wait(DEFAULT_CONFIRMATIONS);
  console.log(`BMCPeriphery: deployed to ${bmcp.address}`);

  console.log(`${target}: management.setBMCPeriphery`);
  await bmcm.setBMCPeriphery(bmcp.address)
    .then((tx) => {
      return tx.wait(DEFAULT_CONFIRMATIONS)
    });
  console.log(`${target}: management.setBMCService`);
  await bmcm.setBMCService(bmcs.address)
    .then((tx) => {
      return tx.wait(DEFAULT_CONFIRMATIONS)
    });
  console.log(`${target}: service.setBMCPeriphery`);
  await bmcs.setBMCPeriphery(bmcp.address)
    .then((tx) => {
      return tx.wait(DEFAULT_CONFIRMATIONS)
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
