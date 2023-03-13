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
      "bsc-docker": {
          url: "http://localhost:8545",
          accounts: [
              // address: 0x59b02D4d2F94ea5c55230715a58EBb0b703bCD4B
              '0xc484de1ef84e998869d59752d1f09bffa161673d54250ea152ec82d684e2f154'
          ]
      },
      "bsc-testnet": {
          url: "https://data-seed-prebsc-1-s1.binance.org:8545",
          accounts: [
              // address: 0x7304b575961F6625ec133ab7e0BCF68dF3cf1044
              '0x389eb48e5d6cc48fde53118da4c1a5e45be98a5a8064e0ceb66a04610cb6d2a0'
          ]
      }
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
