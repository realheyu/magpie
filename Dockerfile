ARG GO_VERSION=1.27.1
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS web-deps
ENV PNPM_HOME=/pnpm
ENV PATH=${PNPM_HOME}:${PATH}
WORKDIR /src/web
RUN corepack enable && corepack prepare pnpm@12.3.4 --activate
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

FROM web-deps AS web-build
COPY web/ ./
RUN pnpm build

FROM golang:${GO_VERSION}-alpine AS go-deps
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download

FROM go-deps AS go-build
COPY . ./
COPY --from=web-build /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/magpie ./cmd/magpie

FROM alpine:3.22 AS runtime
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata && \
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
EXPOSE 8080 8081
ENTRYPOINT ["magpie"]
