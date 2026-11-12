# RepoCortex container image.
#
# Two stages. The first compiles a static binary (CGO disabled) on the build
# machine's own architecture and cross-compiles for the target, so a
# multi-arch build needs no emulation. The second is an empty image holding
# the binary, CA certificates, the licence files and three directories the
# process needs; it has no shell and no package manager.
#
#   docker build -t repocortex .
#   docker run --rm -v "$PWD:/workspace" repocortex index
#
# The index lives in <repository>/.repocortex. To keep the source read-only,
# create that directory first (Docker cannot make a mount point inside a
# read-only mount) and give it a volume of its own:
#
#   mkdir -p .repocortex
#   docker run --rm -v "$PWD:/workspace:ro" -v repocortex-index:/workspace/.repocortex \
#     repocortex index
#
# The release workflow passes GO_VERSION, VERSION, COMMIT and BUILD_TIME. A
# plain `docker build` works without them and stamps the version "dev".

# Keep in step with GO_VERSION in .github/workflows/*.yml and the toolchain
# line in go.mod.
ARG GO_VERSION=1.26.11

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    pkg=github.com/satyendrakumarsingh/repocortex/internal/cli && \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath \
      -ldflags "-s -w -X ${pkg}.Version=${VERSION} -X ${pkg}.Commit=${COMMIT} -X ${pkg}.BuildTime=${BUILD_TIME}" \
      -o /out/repocortex ./cmd/repocortex

# The final image is built FROM scratch and has no shell, so everything it
# needs besides the binary is laid out here and copied across:
#   /tmp                    SQLite and the Go runtime expect a temp directory
#   /home/repocortex        HOME, where user-level configuration is looked up
#   /workspace/.repocortex  the index; a named volume mounted here inherits
#                           this directory's owner, so the non-root user can
#                           write to it while the repository itself is
#                           mounted read-only
RUN mkdir -p /rootfs/tmp /rootfs/home/repocortex /rootfs/workspace/.repocortex /rootfs/etc && \
    chmod 1777 /rootfs/tmp && \
    echo 'repocortex:x:65532:65532:repocortex:/home/repocortex:/sbin/nologin' > /rootfs/etc/passwd && \
    echo 'repocortex:x:65532:' > /rootfs/etc/group && \
    chown -R 65532:65532 /rootfs/home/repocortex /rootfs/workspace

FROM scratch

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

LABEL org.opencontainers.image.title="RepoCortex" \
      org.opencontainers.image.description="Deterministic code intelligence and change safety for AI coding agents: a local code graph served over MCP, a CLI and REST." \
      org.opencontainers.image.url="https://github.com/satyendrakumarsingh/repocortex" \
      org.opencontainers.image.source="https://github.com/satyendrakumarsingh/repocortex" \
      org.opencontainers.image.documentation="https://github.com/satyendrakumarsingh/repocortex/tree/main/docs" \
      org.opencontainers.image.vendor="Satyendra Singh" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_TIME}" \
      io.modelcontextprotocol.server.name="io.github.satyendrakumarsingh/repocortex"

COPY --from=build /rootfs/ /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/repocortex /usr/local/bin/repocortex
COPY LICENSE NOTICE THIRD_PARTY.md /usr/share/doc/repocortex/

ENV HOME=/home/repocortex
USER 65532:65532
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/repocortex"]
CMD ["serve"]
