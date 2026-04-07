# syntax=docker/dockerfile:1.7

FROM golang:1.25.6-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ENV CGO_ENABLED=0

RUN go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/yews \
    ./cmd/yews

FROM python:3.13-slim-bookworm

ENV PIP_NO_CACHE_DIR=1 \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /work

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && python -m pip install --no-cache-dir remarshal

COPY --from=builder /out/yews /usr/local/bin/yews

ENTRYPOINT ["yews"]
CMD ["--help"]

