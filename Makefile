APP_NAME ?= promethius
VERSION_PREFIX ?= vv

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
FRONTEND_REPOSITORY ?= promethius
BACKEND_REPOSITORY ?= employee-service
IMAGE_REPOSITORIES := $(FRONTEND_REPOSITORY) $(BACKEND_REPOSITORY)
NEXT_VERSION := $(shell \
	if command -v $(DOCKER) >/dev/null 2>&1; then \
		$(DOCKER) image ls --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | \
		awk -F: -v prefix='$(VERSION_PREFIX)' -v repos='$(IMAGE_REPOSITORIES)' '\
			BEGIN { split(repos, repo_list, " "); for (i in repo_list) wanted[repo_list[i]] = 1 } \
			wanted[$$1] && $$2 ~ "^" prefix "[0-9]+$$" { \
				n = substr($$2, length(prefix) + 1) + 0; \
				if (n > max) max = n; \
			} \
			END { print prefix (max + 1) }'; \
	else \
		printf '%s1\n' '$(VERSION_PREFIX)'; \
	fi)
VERSION ?= $(NEXT_VERSION)
FRONTEND_IMAGE ?= $(FRONTEND_REPOSITORY):$(VERSION)
BACKEND_IMAGE ?= $(BACKEND_REPOSITORY):$(VERSION)

.PHONY: all build build-frontend build-backend test clean docker-build docker-build-frontend docker-build-backend docker-version help

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

docker-version:
	@printf '%s\n' '$(VERSION)'

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
		'  make docker-version        Print next Docker image tag (vv1, vv2, ...)' \
		'  make docker-build          Build both Docker images with the next vvN tag' \
		'  make docker-build-frontend Build frontend image with the next vvN tag' \
		'  make docker-build-backend  Build backend image with the next vvN tag' \
		'  make clean                 Remove local binaries'
