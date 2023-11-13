SHELL:=/usr/bin/env sh

OSFLAG := ""
UNAME := $(shell uname -m)
ifeq ($(UNAME), arm64)
  OSFLAG = DOCKER_DEFAULT_PLATFORM=linux/amd64
endif

.PHONY:start
start:
	$(OSFLAG) docker-compose -f docker-compose.yml up -d

.PHONY:start-preset
start-preset:
	@if [ ! -d "data" ]; then \
		echo "Extract preset data"; \
		unzip preset.zip; \
	fi
	$(OSFLAG) docker-compose -f docker-compose.yml up -d

.PHONY:stop
stop:
	$(OSFLAG) docker-compose -f docker-compose.yml stop
	docker container prune -f

.PHONY:reset
reset:
	make stop
	rm -rf data logs
	docker system prune -f
