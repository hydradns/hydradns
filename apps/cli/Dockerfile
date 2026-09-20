# hydra CLI + MCP server (multi-arch). Default entrypoint runs `hydra mcp`,
# speaking JSON-RPC 2.0 over stdin/stdout so this image can be listed in the
# official MCP registry (registryType "oci") and run directly by MCP clients:
#   docker run -i --rm -e HYDRA_API_URL -e HYDRA_TOKEN ghcr.io/hydradns/hydra-cli
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETARCH
ARG TARGETOS=linux
# Set by CI to the release tag (e.g. 0.1.0); defaults to "dev" for local/CI
# build-only checks that don't have a version to stamp.
ARG VERSION=dev
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# GOWORK=off matches how .github/workflows/release.yml builds this CLI today
# (the build context here is apps/cli only, so the repo-root go.work isn't
# even present, but this keeps the build environment identical to CI).
ENV GOWORK=off
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w -X github.com/hydradns/hydra-cli/cmd.Version=${VERSION}" \
    -o /bin/hydra .

# Runtime
FROM alpine:3.20
WORKDIR /app

# ca-certificates: HYDRA_API_URL may point at an https:// controlplane.
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/hydra /usr/local/bin/hydra

USER appuser

# Required by the official MCP registry (registryType "oci") to verify the
# publisher owns this image: the value must match server.json's "name" field
# exactly. See docs/modelcontextprotocol-io/package-types.mdx,
# "Docker/OCI Images -> Ownership Verification" in
# github.com/modelcontextprotocol/registry.
LABEL io.modelcontextprotocol.server.name="io.github.hydradns/hydra-mcp"

# Management-API traffic only (stdio MCP, or --http on an operator-chosen
# port); no ports are published by default.
ENTRYPOINT ["/usr/local/bin/hydra"]
CMD ["mcp"]
