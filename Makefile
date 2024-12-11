# Build parameters
BINARY_NAME_SERVER=OpenAuth
BINARY_NAME_CLI=oauthctl
BUILD_DIR=build
GO=go

# Docker parameters
HUB ?= your-registry
TAG ?= $(shell git describe --tags --always --dirty)
DOCKER_IMAGE = ${HUB}/openauth:${TAG}

# Get the current directory
CURRENT_DIR=$(shell pwd)

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT  ?= $(shell git rev-parse --short HEAD)
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
GO_FLAGS=-v
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

# Default target
all: clean build

.PHONY: all clean build build-server build-cli test docker-build docker-push deploy

# Build both binaries
build: build-server build-cli

# Build server
build-server:
	@echo "Building server..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 ${GO} build ${GO_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME_SERVER} ./cmd/OpenAuth

# Build CLI
build-cli:
	@echo "Building CLI..."
	@mkdir -p ${BUILD_DIR}
	${GO} build ${GO_FLAGS} ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME_CLI} ./cmd/oauthctl

# Clean build directory
clean:
	@echo "Cleaning build directory..."
	rm -rf ${BUILD_DIR}

# Run tests
test:
	${GO} test -v ./...

# Install dependencies
deps:
	${GO} mod tidy

# Format code
fmt:
	${GO} fmt ./...

# Run linter
lint:
	golangci-lint run

docker-build:
	@echo "Building Docker image..."
	docker build -t ${DOCKER_IMAGE} -f docker/Dockerfile.OpenAuth .
	docker push ${DOCKER_IMAGE}

docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push ${DOCKER_IMAGE}

# Kubernetes deployment
deploy: docker-push
	@echo "Deploying to Kubernetes..."
	@sed 's|${HUB}/openauth:${TAG}|${DOCKER_IMAGE}|g' configs/deployment/openauth-depl.yaml | kubectl apply -f -

# Show configuration
config:
	@echo "Current configuration:"
	@echo "HUB:          ${HUB}"
	@echo "TAG:          ${TAG}"
	@echo "DOCKER_IMAGE: ${DOCKER_IMAGE}"

# Development targets
run-server: build-server
	@echo "Running server locally..."
	./${BUILD_DIR}/${BINARY_NAME_SERVER}

run-cli: build-cli
	@echo "Running CLI locally..."
	./${BUILD_DIR}/${BINARY_NAME_CLI}