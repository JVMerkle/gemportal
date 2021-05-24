GO = go
GOBUILD = $(GO) build
GOCLEAN=$(GO) clean
GOMOD = $(GO) mod
GOVET=$(GO) vet

GITHASH := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y%m%dT%H%M%SZ')

GOLDFLAGS += -X main.gitHash=$(GITHASH)
GOLDFLAGS += -X main.buildTime=$(BUILDTIME)
GOLDFLAGS += -w -s
GOFLAGS += -ldflags "$(GOLDFLAGS)"

.PHONY: help run build get-deps clean

help:
	@echo "Makefile for gemportal"
	@echo ""
	@echo "Usage:"
	@echo ""
	@echo "	make <commands>"
	@echo ""
	@echo "The commands are:"
	@echo ""
	@echo "	build               Build the package"
	@echo "	clean               Run go clean"
	@echo "	help                Print this help text"
	@echo "	get-deps            Download the dependencies"
	@echo "	vet                 Run go vet"

run: build
	./gemportal

build:
	$(GOBUILD) -o gemportal $(GOFLAGS) .

get-deps:
	$(GOMOD) download

vet:
	$(GOVET) ./...

clean:
	@$(GOCLEAN)