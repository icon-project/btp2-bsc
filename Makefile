SHELL:=/usr/bin/env sh

.PHONY: clean
clean:
	rm -rf build

.PHONY: build-relay
build-relay:
	env GO111MODULE=on go build -o build/bin/relay ./cmd/relay
