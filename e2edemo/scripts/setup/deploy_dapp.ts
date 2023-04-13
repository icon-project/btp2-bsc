import fs from 'fs';
import {ethers} from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {chainType, Deployments} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();

async function deploy_dapp_java(target: string, chain: any) {
  const iconNetwork = IconNetwork.getNetwork(target);
  const dappJar = `${JAVASCORE_PATH}/dapp-sample/build/libs/dapp-sample-0.1.0-optimized.jar`
  const content = fs.readFileSync(dappJar).toString('hex')
  const dapp = new Contract(iconNetwork)
  const deployTxHash = await dapp.deploy({
    content: content,
    params: {
      _callService: chain.contracts.xcall,
    }
  })
  const result = await dapp.getTxResult(deployTxHash)
  if (result.status != 1) {
    throw new Error(`DApp deployment failed: ${result.txHash}`);
  }
  chain.contracts.dapp = dapp.address
  console.log(`${target} DApp: deployed to ${dapp.address}`);
}

async function deploy_dapp_solidity(target: string, chain: any) {
  const DAppSample = await ethers.getContractFactory("DAppProxySample")
  const dappSol = await DAppSample.deploy()
  await dappSol.deployed()
  await dappSol.initialize(chain.contracts.xcall)
  chain.contracts.dapp = dappSol.address
  console.log(`${target} DApp: deployed to ${dappSol.address}`);
}

async function main() {
  const src = deployments.getSrc();
  const dst = deployments.getDst();
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);

  await deploy_dapp_solidity(src, srcChain);
  await deploy_dapp_java(dst, dstChain);

  // update deployments
  deployments.set(src, srcChain);
  deployments.set(dst, dstChain);
  deployments.save();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
