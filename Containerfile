# ---- Builder stage (runs on native platform, cross-compiles for target) ----
FROM --platform=$BUILDPLATFORM registry.access.redhat.com/ubi10/go-toolset:latest AS builder

ARG TARGETARCH=amd64
ARG VERSION=0.0.0-dev

USER root
WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy source code
COPY main.go ./
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY mcp/ mcp/

# Build kubectl-debug-queries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -a \
    -ldflags "-s -w -X github.com/yaacov/kubectl-debug-queries/pkg/version.Version=${VERSION}" \
    -o kubectl-debug-queries \
    main.go

# ---- Runtime stage ----
FROM registry.access.redhat.com/ubi10/ubi-minimal:latest

ARG TARGETARCH=amd64

# Copy binary from builder (set execute permissions during copy)
COPY --from=builder --chmod=755 /build/kubectl-debug-queries /usr/local/bin/kubectl-debug-queries

# --- Environment variables ---
# SSE server settings
ENV MCP_HOST="0.0.0.0" \
    MCP_PORT="8080"

# TLS settings (optional - provide paths to enable TLS)
ENV MCP_CERT_FILE="" \
    MCP_KEY_FILE=""

# Kubernetes authentication (optional - override via HTTP headers in SSE mode)
ENV MCP_KUBE_SERVER="" \
    MCP_KUBE_TOKEN="" \
    MCP_KUBE_INSECURE=""

USER 1001
WORKDIR /home/debug

EXPOSE 8080

ENTRYPOINT ["/bin/sh", "-c", "\
  exec kubectl-debug-queries mcp-server --sse \
    --host \"${MCP_HOST}\" \
    --port \"${MCP_PORT}\" \
    ${MCP_CERT_FILE:+--cert-file \"${MCP_CERT_FILE}\"} \
    ${MCP_KEY_FILE:+--key-file \"${MCP_KEY_FILE}\"} \
    ${MCP_KUBE_SERVER:+--server \"${MCP_KUBE_SERVER}\"} \
    ${MCP_KUBE_TOKEN:+--token \"${MCP_KUBE_TOKEN}\"} \
    $([ \"${MCP_KUBE_INSECURE}\" = \"true\" ] && echo --insecure-skip-tls-verify)"]

LABEL name="kubectl-debug-queries-mcp-server" \
      summary="kubectl-debug-queries MCP server for AI-assisted Kubernetes resource inspection" \
      description="MCP (Model Context Protocol) server exposing Kubernetes resources, logs, and events for AI assistants. Runs in SSE mode over HTTP." \
      io.k8s.display-name="kubectl-debug-queries MCP Server" \
      io.k8s.description="MCP server for kubectl-debug-queries providing AI-assisted Kubernetes resource queries via SSE transport." \
      maintainer="Yaacov Zamir <kobi.zamir@gmail.com>"
