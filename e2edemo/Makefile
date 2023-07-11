PWD = $(abspath ./)

JAVASCORE_PATH = $(PWD)/../javascore
GRADLE = ./gradlew

HARDHAT = npx hardhat
TS_NODE = npx ts-node
NETWORK = --network bsc_localnet
SOLIDITY_CONTRACTS = ./solidity/contracts

.DEFAULT_GOAL := all
all:
	@echo $(PWD)

.PHONY: prepare
prepare:
	npm install

.PHONY: build clean deploy

clean-java:
	(cd $(JAVASCORE_PATH); $(GRADLE) clean)
	$(GRADLE) dapp-sample:clean

clean-solidity:
	$(HARDHAT) clean
	$(RM) -rf \
	    $(SOLIDITY_CONTRACTS)/bmc \
	    $(SOLIDITY_CONTRACTS)/bmv* \
	    $(SOLIDITY_CONTRACTS)/xcall \
	    $(SOLIDITY_CONTRACTS)/../build

clean: clean-java clean-solidity

copy-src:
	@ echo ">>> Copy BMC/BMV/xCall solidity codes" ; \
	$(TS_NODE) scripts/setup/copy_src.ts

compile: copy-src
	@ echo ">>> Compile solidity codes" ; \
	$(HARDHAT) compile

build-bmc-java:
	@ echo ">>> Build BMC javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) bmc:optimizedJar

build-bmv-java:
	@ echo ">>> Build BMV javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) bmv:bsc2:optimizedJar

build-xcall-java:
	@ echo ">>> Build xCall javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) xcall:optimizedJar

build-dapp-java:
	@ echo ">>> Build DApp sample javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) dapp-sample:optimizedJar

build-all: compile build-bmc-java build-bmv-java build-xcall-java build-dapp-java

deploy-bmc:
	@ echo ">>> Deploy BMC contracts" ; \
	E2E_DEMO_PATH=$(PWD) \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_bmc.ts

deploy-bmv:
	@ echo ">>> Deploy BMV contracts" ; \
	E2E_DEMO_PATH=$(PWD) \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_bmv.ts

deploy-xcall:
	@ echo ">>> Deploy xCall contracts" ; \
	E2E_DEMO_PATH=$(PWD) \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_xcall.ts

deploy-dapp:
	@ echo ">>> Deploy DApp contracts" ; \
	JAVASCORE_PATH=$(JAVASCORE_PATH) \
	$(HARDHAT) $(NETWORK) run scripts/setup/deploy_dapp.ts

deploy-all: deploy-bmc deploy-bmv deploy-xcall deploy-dapp

build-bsc:
	@ echo ">>> Build bsc-geth" ; \
	cd docker/bsc2; \
	make build-bsc

start-bsc: build-bsc
	@ echo ">>> Start bsc-geth" ; \
	cd docker/bsc2; \
	make start-preset

start-icon:
	@ echo ">>> Start icon" ; \
	cd docker/icon; \
	make start-preset

start-nodes: start-bsc start-icon
	@ echo ">>> Start nodes" ;

stop-bsc:
	@ echo ">>> Start bsc" ; \
	cd docker/bsc2; \
	make stop

stop-icon:
	@ echo ">>> Stop icon" ; \
	cd docker/icon; \
	make stop

stop-nodes: stop-bsc stop-icon
	@ echo ">>> Stop nodes" ; \

reset-bsc: stop-bsc
	@ echo ">>> Reset bsc" ; \
	cd docker/bsc2; \
	make reset

reset-icon: stop-icon
	@ echo ">>> Reset icon" ; \
	cd docker/icon; \
	make reset

reset-nodes: reset-bsc reset-icon
	@ echo ">>> Reset nodes" ;

setup-icon:
	@ echo ">>> Setup ICON node" ; \
	E2E_DEMO_PATH=$(PWD) \
	$(TS_NODE) scripts/setup/setup_node.ts

run-demo:
	@ echo ">>> Run demo_dapp.ts " ; \
	E2E_DEMO_PATH=$(PWD) \
	$(HARDHAT) $(NETWORK) run scripts/demo_dapp.ts
