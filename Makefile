GO ?= go
PODMAN ?= podman

GITDESCRIBE := $(shell git describe)
BUILDTIME := $(shell date -u '+%Y%m%dT%H%M%SZ')

GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app.gitDescribe=$(GITDESCRIBE)"
GOLDFLAGS += -X "github.com/JVMerkle/gemportal/app.buildTime=$(BUILDTIME)"
GOLDFLAGS += -w -s
GOFLAGS += -ldflags "$(GOLDFLAGS)"

.PHONY: help run build get-deps vet fmt pull-request podman-build podman-run clean

help:
	@echo "Makefile for gemportal"
	@echo ""
	@echo "Usage:"
	@echo ""
	@echo "	make <commands>"
	@echo ""
	@echo "Pass PODMAN=docker to use docker for building."
	@echo ""
	@echo "The commands are:"
	@echo ""
	@echo "	build               Build the package"
	@echo "	clean               Run go clean"
	@echo "	podman-build        Build the gemportal image"
	@echo "	podman-run          Run a gemportal container on port 8080"
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

podman-build:
	$(PODMAN) build -t gemportal .

podman-run: podman-build
	$(PODMAN) run --rm -it -p8080:8080 gemportal

clean:
	@$(GO) clean