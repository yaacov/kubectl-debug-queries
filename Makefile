BINARY  := kubectl-debug-queries
PKG     := github.com/yaacov/kubectl-debug-queries

VERSION_GIT := $(shell git describe --tags 2>/dev/null || echo "0.0.0-dev")
VERSION ?= ${VERSION_GIT}
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG)/pkg/version.Version=$(VERSION) \
	-X $(PKG)/pkg/version.GitCommit=$(GIT_COMMIT) \
	-X $(PKG)/pkg/version.BuildDate=$(BUILD_DATE)

GOOS   := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

IMAGE_REGISTRY ?= quay.io
IMAGE_ORG ?= yaacov
IMAGE_NAME ?= kubectl-debug-queries-mcp-server
IMAGE_TAG ?= $(VERSION)
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_ORG)/$(IMAGE_NAME)
CONTAINER_ENGINE ?= $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)
PROVENANCE_FLAG := $(if $(findstring docker,$(CONTAINER_ENGINE)),--provenance=false,)

.PHONY: help build clean test fmt vet vendor

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## [a-zA-Z0-9_-]+:' $(MAKEFILE_LIST) | sort | \
		awk -F ': ' '{printf "  %-25s %s\n", substr($$1, 4), $$2}'

## build: Build the kubectl-debug-queries binary for current platform
build:
	@echo "Building for ${GOOS}/${GOARCH}"
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(BINARY) .

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -f $(BINARY)-linux-amd64 $(BINARY)-linux-arm64
	rm -f $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64
	rm -f $(BINARY)-windows-amd64.exe
	rm -f $(BINARY)-*-*.tar.gz $(BINARY)-*-*.zip
	rm -f $(BINARY)-*-*.tar.gz.sha256sum $(BINARY)-*-*.zip.sha256sum

## test: Run unit tests
test:
	go test ./...

## fmt: Format Go code
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run go vet and golangci-lint
.PHONY: lint
lint: vet
	$(shell go env GOPATH)/bin/golangci-lint run ./...

## install-tools: Install golangci-lint and other development tools
.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed. Make sure $$(go env GOPATH)/bin is in your PATH."

## vendor: Populate vendor directory
vendor:
	go mod vendor

## build-linux-amd64: Cross-compile for linux/amd64
.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building for linux/amd64"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(BINARY)-linux-amd64 .

## build-linux-arm64: Cross-compile for linux/arm64
.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building for linux/arm64"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -ldflags '$(LDFLAGS)' -o $(BINARY)-linux-arm64 .

## build-darwin-amd64: Cross-compile for darwin/amd64
.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for darwin/amd64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(BINARY)-darwin-amd64 .

## build-darwin-arm64: Cross-compile for darwin/arm64
.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for darwin/arm64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a -ldflags '$(LDFLAGS)' -o $(BINARY)-darwin-arm64 .

## build-windows-amd64: Cross-compile for windows/amd64
.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building for windows/amd64"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags '$(LDFLAGS)' -o $(BINARY)-windows-amd64.exe .

## build-all: Build for all platforms (linux, darwin, windows)
.PHONY: build-all
build-all: clean build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

## dist-all: Create release archives and checksums for all platforms
.PHONY: dist-all
dist-all: build-all
	@echo "Creating release archives..."
	tar -zcvf $(BINARY)-${VERSION}-linux-amd64.tar.gz LICENSE $(BINARY)-linux-amd64
	tar -zcvf $(BINARY)-${VERSION}-linux-arm64.tar.gz LICENSE $(BINARY)-linux-arm64
	tar -zcvf $(BINARY)-${VERSION}-darwin-amd64.tar.gz LICENSE $(BINARY)-darwin-amd64
	tar -zcvf $(BINARY)-${VERSION}-darwin-arm64.tar.gz LICENSE $(BINARY)-darwin-arm64
	zip $(BINARY)-${VERSION}-windows-amd64.zip LICENSE $(BINARY)-windows-amd64.exe
	@echo "Generating checksums..."
	sha256sum $(BINARY)-${VERSION}-linux-amd64.tar.gz > $(BINARY)-${VERSION}-linux-amd64.tar.gz.sha256sum
	sha256sum $(BINARY)-${VERSION}-linux-arm64.tar.gz > $(BINARY)-${VERSION}-linux-arm64.tar.gz.sha256sum
	sha256sum $(BINARY)-${VERSION}-darwin-amd64.tar.gz > $(BINARY)-${VERSION}-darwin-amd64.tar.gz.sha256sum
	sha256sum $(BINARY)-${VERSION}-darwin-arm64.tar.gz > $(BINARY)-${VERSION}-darwin-arm64.tar.gz.sha256sum
	sha256sum $(BINARY)-${VERSION}-windows-amd64.zip > $(BINARY)-${VERSION}-windows-amd64.zip.sha256sum

# ---- Container image targets ----

## image-build-amd64: Build container image for linux/amd64
.PHONY: image-build-amd64
image-build-amd64: vendor
	$(CONTAINER_ENGINE) build \
		--platform linux/amd64 \
		$(PROVENANCE_FLAG) \
		--build-arg TARGETARCH=amd64 \
		--build-arg VERSION=$(VERSION) \
		-f Containerfile \
		-t $(IMAGE):$(IMAGE_TAG)-amd64 \
		.

## image-build-arm64: Build container image for linux/arm64
.PHONY: image-build-arm64
image-build-arm64: vendor
	$(CONTAINER_ENGINE) build \
		--platform linux/arm64 \
		$(PROVENANCE_FLAG) \
		--build-arg TARGETARCH=arm64 \
		--build-arg VERSION=$(VERSION) \
		-f Containerfile \
		-t $(IMAGE):$(IMAGE_TAG)-arm64 \
		.

## image-build-all: Build container images for all architectures
.PHONY: image-build-all
image-build-all: image-build-amd64 image-build-arm64

## image-push-amd64: Push amd64 container image
.PHONY: image-push-amd64
image-push-amd64:
	$(CONTAINER_ENGINE) push $(IMAGE):$(IMAGE_TAG)-amd64

## image-push-arm64: Push arm64 container image
.PHONY: image-push-arm64
image-push-arm64:
	$(CONTAINER_ENGINE) push $(IMAGE):$(IMAGE_TAG)-arm64

## image-manifest: Create and push multi-arch manifest list
.PHONY: image-manifest
image-manifest:
	-@$(CONTAINER_ENGINE) manifest rm $(IMAGE):$(IMAGE_TAG) 2>/dev/null
	$(CONTAINER_ENGINE) manifest create $(IMAGE):$(IMAGE_TAG) \
		$(IMAGE):$(IMAGE_TAG)-amd64 \
		$(IMAGE):$(IMAGE_TAG)-arm64
	$(CONTAINER_ENGINE) manifest push $(IMAGE):$(IMAGE_TAG)
	@echo "Tagging and pushing as latest..."
	-@$(CONTAINER_ENGINE) manifest rm $(IMAGE):latest 2>/dev/null
	$(CONTAINER_ENGINE) manifest create $(IMAGE):latest \
		$(IMAGE):$(IMAGE_TAG)-amd64 \
		$(IMAGE):$(IMAGE_TAG)-arm64
	$(CONTAINER_ENGINE) manifest push $(IMAGE):latest

## image-push-all: Push all arch images and create multi-arch manifest
.PHONY: image-push-all
image-push-all: image-push-amd64 image-push-arm64 image-manifest

# ---- Deploy targets ----

## deploy: Deploy the MCP server pod and service to the current OpenShift cluster
.PHONY: deploy
deploy:
	@echo "Deploying kubectl-debug-queries MCP server..."
	oc apply -f deploy/mcp-server.yaml

## undeploy: Remove the MCP server pod and service from the current OpenShift cluster
.PHONY: undeploy
undeploy:
	@echo "Removing kubectl-debug-queries MCP server..."
	oc delete -f deploy/mcp-server.yaml --ignore-not-found=true

## deploy-route: Expose the MCP server externally via an OpenShift Route
.PHONY: deploy-route
deploy-route:
	@echo "Creating route to expose MCP server..."
	oc apply -f deploy/mcp-route.yaml
	@echo "Route created. Access URL:"
	@oc get route kubectl-debug-queries-mcp-server -n openshift-monitoring -o jsonpath='https://{.spec.host}/sse{"\n"}' 2>/dev/null || echo "  (route not ready yet)"

## undeploy-route: Remove the external route for the MCP server
.PHONY: undeploy-route
undeploy-route:
	@echo "Removing MCP server route..."
	oc delete -f deploy/mcp-route.yaml --ignore-not-found=true

## deploy-olsconfig: Register the MCP server with OpenShift Lightspeed
.PHONY: deploy-olsconfig
deploy-olsconfig:
	@echo "Patching OLSConfig to register kubectl-debug-queries MCP server with Lightspeed..."
	oc patch olsconfig cluster --type merge -p "$$(cat deploy/olsconfig-patch.yaml)"

## undeploy-olsconfig: Unregister the MCP server from OpenShift Lightspeed
.PHONY: undeploy-olsconfig
undeploy-olsconfig:
	@echo "Removing kubectl-debug-queries MCP server from OLSConfig..."
	oc patch olsconfig cluster --type json \
		-p '[{"op":"remove","path":"/spec/mcpServers"},{"op":"remove","path":"/spec/featureGates"}]'

## deploy-all: Deploy the MCP server and register it with Lightspeed
.PHONY: deploy-all
deploy-all: deploy deploy-olsconfig

## undeploy-all: Unregister from Lightspeed and remove the MCP server
.PHONY: undeploy-all
undeploy-all: undeploy-olsconfig undeploy
