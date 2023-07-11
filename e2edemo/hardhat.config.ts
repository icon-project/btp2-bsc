import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  paths: {
    sources: "./solidity/contracts",
    tests: "./solidity/test",
    cache: "./solidity/build/cache",
    artifacts: "./solidity/build/artifacts"
  },
  networks: {
    "bsc_localnet": {
      url: "http://127.0.0.1:8545",
      accounts: [
        // e2e/docker/bsc2
        // address: 0x04d63aBCd2b9b1baa327f2Dda0f873F197ccd186
        "0x59ba8068eb256d520179e903f43dacf6d8d57d72bd306e1bd603fdb8c8da10e8"
      ]
    },
  },
  solidity: {
    version: "0.8.12",
    settings: {
      optimizer: {
        enabled: true,
        runs: 10,
      },
    },
  },
};

export default config;
