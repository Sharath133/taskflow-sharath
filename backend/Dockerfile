# =============================================================================
# TaskFlow API — multi-stage production image (static Go binary on Alpine)
# =============================================================================
# Stage 1 builds a statically linked binary (no libc / CGO) for a minimal
# runtime footprint. Stage 2 ships only Alpine + the binary + non-root user.

# -----------------------------------------------------------------------------
# Builder: compile the API with Go toolchain (pinned per project / requirements)
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS builder

# Install git + ca-certificates so `go mod download` can reach module proxies
# over HTTPS when needed (some indirect deps or GOPROXY setups require it).
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Leverage layer caching: copy module definitions first, then download deps.
# (go.sum is optional; add and commit it for reproducible checksum-verified builds.)
COPY go.mod ./
RUN go mod download

# Copy the rest of the source tree and build a static release binary.
COPY . .

# Target architecture: BuildKit sets TARGETARCH automatically; default keeps
# legacy `docker build` working when the ARG is unset.
ARG TARGETARCH=amd64
# CGO_ENABLED=0 produces a static binary suitable for minimal base images.
# -ldflags "-w -s" strips debug info and the symbol table to shrink size.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-w -s" \
    -o /out/taskflow-api \
    ./cmd/api

# -----------------------------------------------------------------------------
# Runtime: tiny image with non-root execution
# -----------------------------------------------------------------------------
FROM alpine:latest

# Install CA bundle for any outbound TLS (callbacks, OIDC, etc.); keeps image
# small vs full glibc images.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 65532 -S nonroot \
    && adduser -u 65532 -S -G nonroot -h /app -D nonroot

WORKDIR /app

# Copy only the compiled binary from the builder stage.
COPY --from=builder /out/taskflow-api /app/taskflow-api

# Application listens on SERVER_PORT (default 8080); expose for orchestrators.
EXPOSE 8080

# Run as unprivileged user (numeric IDs work everywhere, including Kubernetes).
USER nonroot:nonroot

# Optional: container-level health probe (compose / orchestrator can override).
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health > /dev/null || exit 1

# Run the API process as PID 1 (no shell wrapper).
ENTRYPOINT ["/app/taskflow-api"]
