#!/usr/bin/env bash
# bump-tag.sh — 打 annotated tag 并推送到 origin
#
# 用法:
#   ./tools/bump-tag.sh              # 拉取最新 v* tag，patch +1 后打 tag
#   ./tools/bump-tag.sh v1.0.3       # 直接打指定 tag，不拉取远端最新 tag
#
# 环境变量:
#   REMOTE=origin  TAG_PREFIX=FlashDock
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REMOTE="${REMOTE:-origin}"
TAG_PREFIX="${TAG_PREFIX:-FlashDock}"
EXPLICIT_TAG="${1:-}"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "❌ 当前目录不是 git 仓库" >&2
  exit 1
fi

normalize_tag() {
  local raw="$1"
  raw="$(printf '%s' "$raw" | sed -E 's/^[[:space:]]+|[[:space:]]+$//g')"
  if [[ "$raw" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+].*)?$ ]]; then
    raw="v${raw}"
  fi
  printf '%s' "$raw"
}

validate_tag() {
  local tag="$1"
  local ver="${tag#v}"
  ver="${ver%%[-+]*}"
  if ! [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$ ]]; then
    echo "❌ 非法 tag: ${tag}（期望形如 v1.2.3）" >&2
    return 1
  fi
  return 0
}

if [ -n "$EXPLICIT_TAG" ]; then
  NEW_TAG="$(normalize_tag "$EXPLICIT_TAG")"
  validate_tag "$NEW_TAG"
  echo "📌 使用指定 tag: ${NEW_TAG}（跳过拉取远端最新 tag）"
else
  echo "📥 拉取远程 tags（${REMOTE}）..."
  git fetch --tags --force "$REMOTE"

  LATEST="$(git tag -l 'v[0-9]*' --sort=-v:refname | head -n 1 || true)"
  if [ -z "$LATEST" ]; then
    echo "⚠️ 未找到已有 v* tag，将从 v1.0.0 开始"
    NEW_TAG="v1.0.0"
  else
    echo "📌 当前最新 tag: ${LATEST}"
    VER="${LATEST#v}"
    VER="${VER%%[-+]*}"
    if ! [[ "$VER" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
      echo "❌ 无法解析版本号: ${LATEST}（期望形如 v1.2.3）" >&2
      exit 1
    fi
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
    NEW_TAG="v${MAJOR}.${MINOR}.$((PATCH + 1))"
  fi
fi

if git rev-parse -q --verify "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "❌ 本地已存在 tag ${NEW_TAG}" >&2
  exit 1
fi

if git ls-remote --tags --exit-code "$REMOTE" "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "❌ 远程已存在 tag ${NEW_TAG}" >&2
  exit 1
fi

MSG="${TAG_PREFIX} ${NEW_TAG}"
echo "🏷️  创建 tag: ${NEW_TAG}"
echo "   消息: ${MSG}"
git tag -a "${NEW_TAG}" -m "${MSG}"

echo "🚀 推送 tag 到 ${REMOTE}..."
git push "$REMOTE" "refs/tags/${NEW_TAG}"

echo "✅ 完成: ${NEW_TAG}"
