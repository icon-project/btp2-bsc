SHELL:=/usr/bin/env sh

include .env

.PHONY: relay
relay:
	env GO111MODULE=on go build -o bin/relay ./cmd/relay
	echo "${HOHO}"

.PHONY: clean
clean:
	env GO111MODULE=on go clean -cache
	rm -rf build

.PHONY: compile
compile:
	cd e2edemo && make prepare
