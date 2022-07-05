PROJECT := NetManager
ROOTDIR := $(shell pwd)
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

ORGPATH := github.com/binkynet
REPONAME := $(PROJECT)
REPOPATH := $(ORGPATH)/$(REPONAME)
BINNAME := bnManager

SOURCES := $(shell find . -name '*.go')

.PHONY: all clean deps bootstrap binaries test

all: binaries

clean:
	rm -Rf $(ROOTDIR)/bin

bootstrap:
	go get github.com/mitchellh/gox

binaries: $(SOURCES)
	CGO_ENABLED=0 gox \
		-osarch="linux/amd64 linux/arm darwin/amd64 windows/amd64" \
		-ldflags="-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" \
		-output="bin/{{.OS}}/{{.Arch}}/$(BINNAME)" \
		-tags="netgo" \
		./...

test:
	go test ./...

.PHONY: update-modules
update-modules:
	rm -f go.mod go.sum 
	go mod init github.com/binkynet/NetManager
	go mod edit \
		-replace github.com/coreos/go-systemd=github.com/coreos/go-systemd@e64a0ec8b42a61e2a9801dc1d0abe539dea79197
	go get -u \
		github.com/binkynet/BinkyNet@2c1cc06db2b5e9bcb533366efea36298aaabc8ac
	go mod tidy
