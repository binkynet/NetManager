PROJECT := NetManager
ROOTDIR := $(shell pwd)
VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD)

GOBUILDDIR := $(ROOTDIR)/.gobuild
VENDORDIR := $(ROOTDIR)/vendor

ORGPATH := github.com/binkynet
ORGDIR := $(GOBUILDDIR)/src/$(ORGPATH)
REPONAME := $(PROJECT)
REPODIR := $(ORGDIR)/$(REPONAME)
REPOPATH := $(ORGPATH)/$(REPONAME)
BINNAME := bnManager

GOPATH := $(GOBUILDDIR)
GOVERSION := 1.9.0-alpine

ifndef GOOS
	GOOS := $(shell go env GOHOSTOS)
endif
ifndef GOARCH
	GOARCH := $(shell go env GOHOSTARCH)
endif
ifndef GOEXE
	GOEXE := $(shell go env GOHOSTEXE)
endif

BINPATH := bin/$(GOOS)/$(GOARCH)
BINDIR := $(ROOTDIR)/$(BINPATH)
BIN := $(BINDIR)/$(BINNAME)$(GOEXE)

SOURCES := $(shell find . -name '*.go')

.PHONY: all clean deps

all: local 

build: $(BIN)

clean:
	rm -Rf $(BIN) $(ROOTDIR)/$(BINNAME) $(ROOTDIR)/bin $(GOBUILDDIR)

deps:
	@${MAKE} -B -s .gobuild

local:
	@${MAKE} build
	@ln -sf $(BIN) $(ROOTDIR)/$(BINNAME)

binaries:
	@${MAKE} -B GOOS=linux GOARCH=amd64 build
	@${MAKE} -B GOOS=linux GOARCH=arm build
	@${MAKE} -B GOOS=darwin GOARCH=amd64 build
	@${MAKE} -B GOOS=windows GOARCH=amd64 GOEXE=.exe build

.gobuild:
	@mkdir -p $(ORGDIR)
	@rm -f $(REPODIR) && ln -s ../../../../ $(REPODIR)
	@GOPATH=$(GOPATH) pulsar go flatten -V $(VENDORDIR)
	@GOPATH=$(GOPATH) pulsar go get $(ORGPATH)/BinkyNet/...

$(BIN): .gobuild $(SOURCES)
	docker run \
		--rm \
		-v $(ROOTDIR):/usr/code \
		-e GOPATH=/usr/code/.gobuild \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e CGO_ENABLED=0 \
		-w /usr/code/ \
		golang:$(GOVERSION) \
		go build -a -ldflags "-X main.projectVersion=$(VERSION) -X main.projectBuild=$(COMMIT)" -o $(BINPATH)/$(BINNAME)$(GOEXE) $(REPOPATH)

test: $(BIN)
	go test ./...

