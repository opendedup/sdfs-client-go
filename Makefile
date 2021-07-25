PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)

GOARCH := $(shell go env GOARCH)
GOOS := $(shell go env GOOS)

VERSION ?= $(shell git describe --tags)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
HASH := $(shell git rev-parse HEAD)
TAG ?= "sdfs/sdfscli:$(VERSION)"

all: build

checks:
	@echo "Checking dependencies"
	@(env bash $(PWD)/buildscripts/checkdeps.sh)

getdeps:
	@mkdir -p ${GOPATH}/bin
	@which golangci-lint 1>/dev/null || (echo "Installing golangci-lint" && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.27.0)

crosscompile:
	@(env bash $(PWD)/buildscripts/cross-compile.sh)

verifiers: getdeps fmt lint

fmt:
	@echo "Running $@ check"
	@GO111MODULE=on gofmt -d cmd/
	@GO111MODULE=on gofmt -d pkg/

lint:
	@echo "Running $@ check"
	@GO111MODULE=on ${GOPATH}/bin/golangci-lint cache clean
	@GO111MODULE=on ${GOPATH}/bin/golangci-lint run --timeout=5m --config ./.golangci.yml



# Builds sdfscli locally.
build:
	@mkdir -p $(PWD)/build
	@echo "Building sdfscli binary to '$(PWD)/build/sdfscli'"
	@go build -ldflags="-X 'main.Version=$(BRANCH)' -X 'main.BuildDate=$$(date -Iseconds)'" -o ./build/sdfscli app/*
	@go build -ldflags="-X 'main.Version=$(BRANCH)' -X 'main.BuildDate=$$(date -Iseconds)'" -o ./build/gentool gentool/*
	@mv utils/utils_linux.go .
	@env GOARCH=amd64 GOOS=windows go build -ldflags="-X 'main.Version=$(BRANCH)' -X 'main.BuildDate=$$(date -Iseconds)'" -o ./build/sdfscli.exe app/*
	@env GOARCH=amd64 GOOS=windows go build -ldflags="-X 'main.Version=$(BRANCH)' -X 'main.BuildDate=$$(date -Iseconds)'" -o ./build/gentool.exe gentool/*
	@mv utils_linux.go utils/

# Builds sdfscli and installs it to $GOPATH/bin.
install: build
	@echo "Installing sdfscli binary to '$(GOPATH)/bin/sdfscli'"
	@mkdir -p $(GOPATH)/bin && cp -f $(PWD)/build/sdfscli $(GOPATH)/bin/sdfscli
	@echo "Installation successful. To learn more, try \"sdfscli\"."

clean:
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf sdfscli
	@rm -rvf build
	@rm -rvf release
	@rm -rvf .verify*
