APP_NAME ?= promethius
VERSION ?= latest

GO ?= go
GOOS ?= linux
GOARCH ?= amd64
GOCACHE ?= $(CURDIR)/.cache/go-build
CGO_ENABLED ?= 0
GO_BUILD_FLAGS ?= -buildvcs=false -trimpath

BIN_DIR ?= bin
FRONTEND_BIN := $(BIN_DIR)/frontend
BACKEND_BIN := $(BIN_DIR)/backend

DOCKER ?= docker
DOCKERFILE ?= Dockerfile
FRONTEND_IMAGE ?= promethius:$(VERSION)
BACKEND_IMAGE ?= employee-service:$(VERSION)

.PHONY: all build build-frontend build-backend test clean docker-build docker-build-frontend docker-build-backend help

all: build

build: build-frontend build-backend

build-frontend:
	@mkdir -p $(BIN_DIR) $(GOCACHE)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOCACHE=$(GOCACHE) \
		$(GO) build $(GO_BUILD_FLAGS) -o $(FRONTEND_BIN) ./cmd/frontend

build-backend:
	@mkdir -p $(BIN_DIR) $(GOCACHE)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOCACHE=$(GOCACHE) \
		$(GO) build $(GO_BUILD_FLAGS) -o $(BACKEND_BIN) ./cmd/backend

test:
	GOCACHE=$(GOCACHE) $(GO) test -buildvcs=false ./...

clean:
	rm -rf $(BIN_DIR)

docker-build: docker-build-frontend docker-build-backend

docker-build-frontend:
	$(DOCKER) build \
		-f $(DOCKERFILE) \
		--build-arg SERVICE=frontend \
		-t $(FRONTEND_IMAGE) .

docker-build-backend:
	$(DOCKER) build \
		-f $(DOCKERFILE) \
		--build-arg SERVICE=backend \
		-t $(BACKEND_IMAGE) .

help:
	@printf '%s\n' \
		'Targets:' \
		'  make build                 Build frontend and backend binaries' \
		'  make build-frontend        Build bin/frontend' \
		'  make build-backend         Build bin/backend' \
		'  make test                  Run Go tests' \
		'  make docker-build          Build both Docker images' \
		'  make docker-build-frontend Build frontend image' \
		'  make docker-build-backend  Build backend image' \
		'  make clean                 Remove local binaries'
