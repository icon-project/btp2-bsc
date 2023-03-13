PWD = $(abspath ./)

JAVASCORE_PATH = $(PWD)/../javascore
GRADLE = ./gradlew

HARDHAT = npx hardhat
TS_NODE = npx ts-node
NETWORK = --network bsc-docker
SOLIDITY_CONTRACTS = ./solidity/contracts

.DEFAULT_GOAL := all
all:
	@echo $(PWD)

.PHONY: prepare
prepare:
	npm install

.PHONY: custom-cmd
custom-cmd:
	@ echo ">>> HI~"

.PHONY: build clean deploy

clean-java:
	(cd $(JAVASCORE_PATH); $(GRADLE) clean)
	$(GRADLE) dapp-sample:clean

clean-solidity:
	$(HARDHAT) clean
	$(RM) -rf \
	    $(SOLIDITY_CONTRACTS)/bmc \
	    $(SOLIDITY_CONTRACTS)/bmv* \
	    $(SOLIDITY_CONTRACTS)/xcall

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
	$(GRADLE) bmv:bsc:optimizedJar

build-xcall-java:
	@ echo ">>> Build xCall javascore" ; \
	cd $(JAVASCORE_PATH) ; \
	$(GRADLE) xcall:optimizedJar

build-dapp-java:
	@ echo ">>> Build DApp sample javascore" ; \
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
	E2E_DEMO_PATH=$(PWD) \
	$(HARDHAT) $(NETWORK) run scripts/deploy_dapp.ts

deploy-all: deploy-bmc deploy-bmv deploy-xcall deploy-dapp

build-bsc:
	@ echo ">>> Build bsc-geth" ; \
	cd docker/bsc; \
	make build-simple

bootstrap-bsc: build-bsc
	@ echo ">>> Bootstrap bsc-geth" ; \
	cd docker/bsc; \
	make bootstrap-simple

start-bsc:
	@ echo ">>> Start bsc-geth" ; \
	cd docker/bsc; \
	make start-simple

start-nodes: start-bsc
	@ echo ">>> Start nodes" ; \
	cd docker; \
	docker compose up icon-node -d

stop-bsc:
	@ echo ">>> Start nodes" ; \
	cd docker/bsc; \
	make stop-all

reset-bsc:
	@ echo ">>> Start nodes" ; \
	cd docker/bsc; \
	make reset

stop-nodes: stop-bsc reset-bsc
	@ echo ">>> Stop nodes" ; \
	cd docker; \
	docker compose down

setup-node:
	@ echo ">>> Setup ICON node" ; \
	E2E_DEMO_PATH=$(PWD) \
	$(TS_NODE) scripts/setup/setup_node.ts

run-demo:
	@ echo ">>> Run demo_dapp.ts " ; \
	E2E_DEMO_PATH=$(PWD) \
	$(HARDHAT) $(NETWORK) run scripts/demo_dapp.ts
