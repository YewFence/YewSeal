# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS mise-base

WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN curl https://mise.run | sh

ENV PATH=/root/.local/bin:$PATH \
    MISE_CACHE_DIR=/mise/cache \
    MISE_DATA_DIR=/mise/data \
    MISE_LOCKED=1 \
    MISE_TASK_RUN_AUTO_INSTALL=false

FROM mise-base AS yews-builder

COPY mise.toml mise.lock ./
RUN --mount=type=cache,target=/mise/cache \
    mise trust mise.toml && mise install go

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    mise exec -- go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} mise exec -- go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/yews \
    ./cmd/yews

FROM mise-base AS remarshal-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    pipx \
    python3 \
    && rm -rf /var/lib/apt/lists/*

COPY mise.toml mise.lock ./
RUN --mount=type=cache,target=/mise/cache \
    mise trust mise.toml && mise install pipx:remarshal

RUN mkdir -p /usr/local/remarshal/bin \
    && ln -s "$(mise where pipx:remarshal)/bin/toml2yaml" /usr/local/remarshal/bin/toml2yaml \
    && ln -s "$(mise where pipx:remarshal)/bin/yaml2toml" /usr/local/remarshal/bin/yaml2toml \
    && ln -s "$(mise where pipx:remarshal)/bin/remarshal" /usr/local/remarshal/bin/remarshal

FROM debian:bookworm-slim AS full

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    python3 \
    && rm -rf /var/lib/apt/lists/*

ENV PATH=/usr/local/remarshal/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /work

COPY --from=remarshal-builder /mise/data/installs /mise/data/installs
COPY --from=remarshal-builder /usr/local/remarshal /usr/local/remarshal
COPY --from=yews-builder /out/yews /usr/local/bin/yews

ENTRYPOINT ["yews"]
CMD ["--help"]

FROM alpine:3.22 AS certs

RUN apk add --no-cache ca-certificates

FROM scratch AS lite

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

WORKDIR /work

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=yews-builder /out/yews /yews

ENTRYPOINT ["/yews"]
CMD ["--help"]
