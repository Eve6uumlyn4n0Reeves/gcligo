FROM golang:1.22 as build
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
# Build with target platform (defaults to linux/amd64 if not provided)
RUN set -eux; \
    GOOS=${TARGETOS:-linux}; \
    GOARCH=${TARGETARCH:-amd64}; \
    echo "building for $GOOS/$GOARCH"; \
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -o gcli2api-go ./cmd/server
# Prepare runtime assets and writable dirs
RUN mkdir -p /data/auths /data/storage

FROM busybox:1.36.1-uclibc AS bb

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/gcli2api-go /app/gcli2api-go
COPY --from=build /app/web /app/web
# Provide a default config inside the image (can be overridden by mounting /app/config.yaml)
COPY --from=build /app/config.docker.yaml /app/config.yaml
# Provide writable data directories (owned by nonroot: 65532)
COPY --from=build --chown=65532:65532 /data /data
# Ensure CA certificates are present for outgoing TLS
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# Provide a tiny toolbox for HEALTHCHECK without increasing attack surface
COPY --from=bb /bin/busybox /busybox
# Set defaults for new features; override via env as needed
ENV OPENAI_PORT=8317
ENV AUTH_DIR=/data/auths \
    HEADER_PASSTHROUGH=false \
    AUTO_IMAGE_PLACEHOLDER=true
EXPOSE 8317 8318
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
  CMD ["/busybox","sh","-c","wget -q -O- http://127.0.0.1:${OPENAI_PORT}/healthz >/dev/null 2>&1 || exit 1"]
# Run as non-root (65532:65532 is nonroot user in distroless)
USER 65532:65532
ENTRYPOINT ["/app/gcli2api-go"]
