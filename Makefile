#!/usr/bin/env bash
# Makefile with some common workflow for dev, build and test

.PHONY: test
test: ## Test all the sub-packages, uses: go test -v $(go list ./...)
	GO111MODULE=on TRACE=1 go test -v ./

install: build-local
	cp bin/sema /usr/local/bin/sema

.PHONY: build-run-test
build-local: ## Builds using your local Golang installation
	GO111MODULE=on go build -o bin/sema ./

.PHONY: clean
clean: ## Deletes all locally build binaries again
	rm bin/sema
