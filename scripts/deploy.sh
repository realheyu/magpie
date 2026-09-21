#!/usr/bin/env bash
# 更新当前仓库并构建 Magpie 镜像。
#
# 第一次部署请先手动 clone 仓库，然后在仓库根目录执行：
#   ./scripts/deploy.sh v1.0.0
#
# 可选参数：
#   ./scripts/deploy.sh v1.0.0 --image registry.example.com/magpie
#   ./scripts/deploy.sh v1.0.0 --platform linux/arm64
#   ./scripts/deploy.sh v1.0.0 --branch main --remote origin
#   ./scripts/deploy.sh v1.0.0 --no-update       # 只构建当前 checkout
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REMOTE="${REMOTE:-}"
BRANCH="${BRANCH:-}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-magpie}"
PLATFORM="${PLATFORM:-linux/amd64}"
UPDATE=1

usage() {
  cat <<'EOF'
用法：scripts/deploy.sh <版本号> [选项]

更新仓库后构建指定版本的镜像。镜像默认为 magpie:<版本号>，构建后会额外生成
magpie:<版本号>-git-<commit> 标签，方便确认版本和回滚。

选项：
  --image REPOSITORY  镜像仓库名称（默认 magpie）
  --platform PLATFORM Docker 目标平台（默认 linux/amd64）
  --remote NAME       Git 远程仓库（默认使用当前分支 upstream，否则 origin）
  --branch NAME       要更新的分支（默认当前分支）
  --no-update         跳过 git 更新，只构建当前代码
  -h, --help          显示帮助

示例：
  scripts/deploy.sh v1.0.0
  scripts/deploy.sh 20260921 --image registry.example.com/magpie
  PLATFORM=linux/arm64 scripts/deploy.sh v1.0.0
EOF
}

die() {
  echo "错误：$*" >&2
  exit 1
}

if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
  usage
  exit 0
fi

[[ $# -ge 1 ]] || die "必须传版本号，例如：scripts/deploy.sh v1.0.0"
VERSION="$1"
shift
[[ "$VERSION" != -* ]] || die "版本号必须作为第一个参数传入，例如：scripts/deploy.sh v1.0.0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --image)
      [[ $# -ge 2 ]] || die "--image 需要参数"
      IMAGE_REPOSITORY="$2"
      shift 2
      ;;
    --platform)
      [[ $# -ge 2 ]] || die "--platform 需要参数"
      PLATFORM="$2"
      shift 2
      ;;
    --remote)
      [[ $# -ge 2 ]] || die "--remote 需要参数"
      REMOTE="$2"
      shift 2
      ;;
    --branch)
      [[ $# -ge 2 ]] || die "--branch 需要参数"
      BRANCH="$2"
      shift 2
      ;;
    --no-update)
      UPDATE=0
      shift
      ;;
    -*)
      die "未知参数：$1（使用 --help 查看用法）"
      ;;
    *)
      die "只能传一个版本号：$VERSION"
      ;;
  esac
done

[[ "$VERSION" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] || \
  die "版本号不符合 Docker 标签格式：$VERSION"

command -v git >/dev/null 2>&1 || die "未找到 git"
command -v docker >/dev/null 2>&1 || die "未找到 docker"
git -C "$ROOT" rev-parse --show-toplevel >/dev/null 2>&1 || die "脚本必须位于 Git 仓库中"

cd "$ROOT"

if [[ "$UPDATE" -eq 1 ]]; then
  # 只更新干净的源码 checkout；被 .gitignore 忽略的 config.toml 不受影响。
  git diff --quiet || die "工作区有未提交修改，请先提交或 stash 后再更新"
  git diff --cached --quiet || die "暂存区有未提交修改，请先提交或 stash 后再更新"

  CURRENT_BRANCH="$(git symbolic-ref --quiet --short HEAD || true)"
  [[ -n "$CURRENT_BRANCH" ]] || die "当前处于 detached HEAD，请先切换到要部署的分支"
  if [[ -z "$BRANCH" ]]; then
    BRANCH="$CURRENT_BRANCH"
  fi
  [[ "$BRANCH" == "$CURRENT_BRANCH" ]] || die "当前分支为 $CURRENT_BRANCH，请先切换到 $BRANCH"

  if [[ -z "$REMOTE" ]]; then
    REMOTE="$(git config --get "branch.${BRANCH}.remote" || true)"
    REMOTE="${REMOTE:-origin}"
  fi
  git remote get-url "$REMOTE" >/dev/null 2>&1 || die "Git 远程仓库不存在：$REMOTE"

  echo "==> 更新仓库：${REMOTE}/${BRANCH}"
  git fetch --prune "$REMOTE" "$BRANCH"
  git merge --ff-only "${REMOTE}/${BRANCH}"
else
  echo "==> 跳过仓库更新"
fi

COMMIT="$(git rev-parse --short HEAD)"
IMAGE="${IMAGE_REPOSITORY}:${VERSION}"
GIT_IMAGE="${IMAGE_REPOSITORY}:${VERSION}-git-${COMMIT}"

echo "==> 构建镜像：${IMAGE}（${PLATFORM}，commit ${COMMIT}）"
docker build --pull --platform "$PLATFORM" -t "$IMAGE" .
docker tag "$IMAGE" "$GIT_IMAGE"

echo "==> 构建完成"
echo "    image:  $IMAGE"
echo "    commit: $GIT_IMAGE"
