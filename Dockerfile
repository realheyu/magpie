# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.1
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS web-deps
ENV PNPM_HOME=/pnpm
ENV PATH=${PNPM_HOME}:${PATH}
WORKDIR /src/web
RUN corepack enable && corepack prepare pnpm@12.3.4 --activate
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN --mount=type=cache,id=magpie-pnpm,target=/pnpm/store \
    pnpm config set store-dir /pnpm/store && \
    pnpm install --frozen-lockfile

FROM web-deps AS web-build
COPY web/ ./
RUN pnpm build

FROM golang:${GO_VERSION}-alpine AS go-deps
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM go-deps AS go-build
COPY . ./
COPY --from=web-build /src/web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/magpie ./cmd/magpie

FROM alpine:3.24 AS runtime
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S magpie && \
    adduser -S -G magpie magpie && \
    mkdir -p /app/logs /etc/magpie && \
    chown -R magpie:magpie /app /etc/magpie

ENV TZ=Asia/Shanghai
ENV MAGPIE_CONFIG_FILE=/etc/magpie/config.toml

WORKDIR /app
COPY --from=go-build /out/magpie /usr/local/bin/magpie
COPY config.example.toml /etc/magpie/config.example.toml

USER magpie
EXPOSE 6080 6081
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD wget -qO- http://127.0.0.1:6081/healthz >/dev/null || exit 1
ENTRYPOINT ["magpie"]
