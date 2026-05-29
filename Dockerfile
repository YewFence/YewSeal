# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS yews-builder

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
    MISE_TASK_RUN_AUTO_INSTALL=false

COPY mise.toml ./
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

FROM alpine:3.22 AS remarshal-builder

RUN apk add --no-cache python3 py3-pip

RUN python3 -m venv /opt/remarshal-env

ARG REMARSHAL_VERSION=1.3.0

ENV PATH=/opt/remarshal-env/bin:$PATH \
    PIP_NO_CACHE_DIR=1 \
    PYTHONDONTWRITEBYTECODE=1

RUN pip install --no-cache-dir "remarshal==${REMARSHAL_VERSION}"

FROM alpine:3.22 AS full

RUN apk add --no-cache ca-certificates python3

ENV PATH=/opt/remarshal-env/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /work

COPY --from=remarshal-builder /opt/remarshal-env /opt/remarshal-env
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
