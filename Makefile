#!/usr/bin/env bash
# Makefile with some common workflow for dev, build and test

.PHONY: test
test: ## Test all the sub-packages, uses: go test -v $(go list ./...)
	GO111MODULE=on TRACE=1 go test -cover -v ./

install: build-local
	cp dist/sema /usr/local/bin/sema

.PHONY: build-run-test
build-local: ## Builds using your local Golang installation
	GO111MODULE=on CGO_ENABLED=0 go build -ldflags="-w -s" -o dist/sema ./
# -ldflags + CGO: https://stackoverflow.com/questions/55106186/no-such-file-or-directory-with-docker-scratch-image

.PHONY: test-release
test-release: ## Performs a local release sequence
	goreleaser --snapshot --skip-publish --rm-dist


.PHONY: clean
clean: ## Deletes all locally build binaries again
	rm dist/sema
