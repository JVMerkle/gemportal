GO ?= go
DOCKER ?= docker

GOBUILD = $(GO) build
GOCLEAN =$(GO) clean
GOMOD = $(GO) mod
GOVET =$(GO) vet

GITHASH := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y%m%dT%H%M%SZ')

GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app/cfg.gitHash=$(GITHASH)"
GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app/cfg.buildTime=$(BUILDTIME)"
GOLDFLAGS += -w -s
GOFLAGS += -ldflags "$(GOLDFLAGS)"

.PHONY: help run build get-deps vet docker-build docker-run clean

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
	@echo "	docker-build        Build the gemportal image"
	@echo "	docker-run          Run a gemportal container on port 8080"
	@echo "	get-deps            Download the dependencies"
	@echo "	help                Print this help text"
	@echo "	vet                 Run go vet"

run: build
	./gemportal

build:
	$(GOBUILD) -o gemportal $(GOFLAGS) .

get-deps:
	$(GOMOD) download

vet:
	$(GOVET) ./...

docker-build:
	$(DOCKER) build -t gemportal .

docker-run: docker-build
	$(DOCKER) run --rm -it -p8080:8080 gemportal

clean:
	@$(GOCLEAN)