import {ethers} from 'hardhat';
import {Contract, IconNetwork, Jar} from "../icon";
import {chainType, Deployments, DEFAULT_CONFIRMATIONS} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();

async function deploy_dapp_java(target: string, chain: any) {
  const iconNetwork = IconNetwork.getNetwork(target);
  const content = Jar.readFromFile(JAVASCORE_PATH, "dapp-sample");
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
  await dappSol.deployTransaction.wait(DEFAULT_CONFIRMATIONS);
  console.log('xcall addr:', chain.contracts.xcall);
  await (await dappSol.initialize(chain.contracts.xcall)).wait(DEFAULT_CONFIRMATIONS);
  chain.contracts.dapp = dappSol.address
  console.log(`${target} DApp: deployed to ${dappSol.address}`);
}

async function main() {
  const src = deployments.getSrc();
  const dst = deployments.getDst();
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);

  // deploy to src network first
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
