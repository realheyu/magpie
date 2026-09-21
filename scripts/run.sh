#!/usr/bin/env bash
# 停止并删除旧容器，然后以 host 网络启动指定版本的 Magpie 镜像。
set -euo pipefail

usage() {
  echo "用法：scripts/run.sh <版本号>" >&2
}

if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
  usage
  exit 0
fi

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
if [[ ! "$VERSION" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]]; then
  echo "版本号不符合 Docker 标签格式：$VERSION" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="magpie:$VERSION"
CONTAINER="magpie"

docker stop "$CONTAINER" >/dev/null 2>&1 || true
docker rm "$CONTAINER" >/dev/null 2>&1 || true

docker run -d \
  --name "$CONTAINER" \
  --restart unless-stopped \
  --network host \
  -v "$ROOT/config.toml:/etc/magpie/config.toml:ro" \
  "$IMAGE"
