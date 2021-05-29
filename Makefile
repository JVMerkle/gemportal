GO ?= go
DOCKER ?= docker

GITHASH := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y%m%dT%H%M%SZ')

GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app.gitHash=$(GITHASH)"
GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app.buildTime=$(BUILDTIME)"
GOLDFLAGS += -w -s
GOFLAGS += -ldflags "$(GOLDFLAGS)"

.PHONY: help run build get-deps vet fmt pull-request docker-build docker-run clean

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
	@echo "	fmt                 Run go fmt"
	@echo "	get-deps            Download the dependencies"
	@echo "	help                Print this help text"
	@echo "	pull-request        Check if you changes are ready to be submitted"
	@echo "	vet                 Run go vet"

run: build
	./gemportal

build:
	$(GO) build -o gemportal $(GOFLAGS) .

get-deps:
	$(GO) mod download

vet:
	$(GO) vet ./...
	
fmt:
	$(GO) fmt ./...

pull-request: build fmt vet
	@echo "Your code looks good!"

docker-build:
	$(DOCKER) build -t gemportal .

docker-run: docker-build
	$(DOCKER) run --rm -it -p8080:8080 gemportal

clean:
	@$(GO) clean