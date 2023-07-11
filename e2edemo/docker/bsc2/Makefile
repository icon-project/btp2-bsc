SHELL:=/usr/bin/env sh

OSFLAG := ""
UNAME := $(shell uname -m)
ifeq ($(UNAME), arm64)
  OSFLAG = DOCKER_DEFAULT_PLATFORM=linux/amd64
endif

.PHONY:build-bsc
build-bsc:
	$(OSFLAG) docker-compose -f docker-compose.bsc.yml build

.PHONY:build-bootstrap
build-bootstrap:
	$(OSFLAG) docker-compose -f docker-compose.bootstrap.yml build

.PHONY:build
build: build-bsc build-bootstrap
	$(OSFLAG) docker-compose -f docker-compose.yml build

.PHONY:bootstrap
bootstrap:
	$(OSFLAG) docker-compose -f docker-compose.bootstrap.yml run bootstrap

.PHONY:start
start:
	$(OSFLAG) docker-compose -f docker-compose.yml up -d

.PHONY:start-preset
start-preset:
	@if [ ! -d "data" ]; then \
		echo "Extract preset data"; \
		unzip -d storage preset.zip; \
	fi
	$(OSFLAG) docker-compose -f docker-compose.yml up -d

.PHONY:stop
stop:
	docker-compose -f docker-compose.yml stop
	docker container prune -f

.PHONY:reset
reset:
	make stop
	rm -rf storage
	docker system prune -f
