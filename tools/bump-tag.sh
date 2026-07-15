#!/usr/bin/env bash
# bump-tag.sh — 同步本地版本号、打 annotated tag 并推送到 origin
#
# 用法:
#   ./tools/bump-tag.sh              # 拉取最新 v* tag，patch +1
#   ./tools/bump-tag.sh v1.0.3       # 直接使用指定版本
#
# 流程:
#   1. 解析 NEW_TAG（如 v1.0.4）
#   2. 写回 app/version.go、wails.json（与 tag 一致，不含 v 前缀）
#   3. 若有变更则 commit
#   4. 在该 commit 上打 tag 并 push（含 commit + tag）
#
# 环境变量:
#   REMOTE=origin  TAG_PREFIX=FlashDock  SKIP_COMMIT=1（只改文件不提交，也不打 tag）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REMOTE="${REMOTE:-origin}"
TAG_PREFIX="${TAG_PREFIX:-FlashDock}"
EXPLICIT_TAG="${1:-}"
SKIP_COMMIT="${SKIP_COMMIT:-0}"

VERSION_GO="app/version.go"
WAILS_JSON="wails.json"

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
  if ! [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$ ]]; then
    echo "❌ 非法 tag: ${tag}（期望形如 v1.2.3）" >&2
    return 1
  fi
  return 0
}

# 将 NEW_TAG 同步到本地源码（开发机 / wails build 默认读这里）
sync_version_files() {
  local tag="$1"
  local ver="${tag#v}"
  ver="${ver%%[-+]*}"

  if [ ! -f "$VERSION_GO" ]; then
    echo "❌ 找不到 ${VERSION_GO}" >&2
    exit 1
  fi
  if [ ! -f "$WAILS_JSON" ]; then
    echo "❌ 找不到 ${WAILS_JSON}" >&2
    exit 1
  fi

  # app/version.go: var Version = "x.y.z"
  if grep -Eq '^var Version = "' "$VERSION_GO"; then
    sed -E -i.bak 's/^var Version = "[^"]*"/var Version = "'"${ver}"'"/' "$VERSION_GO"
    rm -f "${VERSION_GO}.bak"
  else
    echo "❌ ${VERSION_GO} 中未找到 var Version = \"...\"" >&2
    exit 1
  fi

  # wails.json info.productVersion
  if command -v python3 >/dev/null 2>&1; then
    APP_VERSION="$ver" python3 - <<'PY'
import json, os
path = "wails.json"
ver = os.environ["APP_VERSION"]
with open(path, encoding="utf-8") as f:
    cfg = json.load(f)
cfg.setdefault("info", {})["productVersion"] = ver
with open(path, "w", encoding="utf-8") as f:
    json.dump(cfg, f, indent=4, ensure_ascii=False)
    f.write("\n")
print(f"wails.json productVersion -> {ver}")
PY
  elif command -v node >/dev/null 2>&1; then
    APP_VERSION="$ver" node -e '
const fs = require("fs");
const path = "wails.json";
const ver = process.env.APP_VERSION;
const cfg = JSON.parse(fs.readFileSync(path, "utf8"));
cfg.info = cfg.info || {};
cfg.info.productVersion = ver;
fs.writeFileSync(path, JSON.stringify(cfg, null, 4) + "\n");
console.log("wails.json productVersion -> " + ver);
'
  else
    echo "❌ 需要 python3 或 node 来更新 wails.json" >&2
    exit 1
  fi

  echo "📝 本地版本已同步: ${ver}"
  echo "   - ${VERSION_GO}"
  echo "   - ${WAILS_JSON}"
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

APP_VERSION="${NEW_TAG#v}"
APP_VERSION="${APP_VERSION%%[-+]*}"
export APP_VERSION

sync_version_files "$NEW_TAG"

if [ "$SKIP_COMMIT" = "1" ]; then
  echo "⏭️  SKIP_COMMIT=1，已写回版本文件，跳过 commit / tag / push"
  exit 0
fi

# 若工作区有未提交改动（版本文件），先提交版本号
if ! git diff --quiet -- "$VERSION_GO" "$WAILS_JSON"; then
  git add -- "$VERSION_GO" "$WAILS_JSON"
  git commit -m "chore: bump version to ${APP_VERSION}"
  echo "✅ 已提交版本号变更: ${APP_VERSION}"
else
  echo "ℹ️  版本文件已是 ${APP_VERSION}，无需额外 commit"
fi

if git rev-parse -q --verify "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "🗑️  删除本地已有 tag ${NEW_TAG}"
  git tag -d "${NEW_TAG}" >/dev/null 2>&1 || true
fi

if git ls-remote --tags --exit-code "$REMOTE" "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "🗑️  删除远程已有 tag ${NEW_TAG}"
  git push "$REMOTE" ":refs/tags/${NEW_TAG}" >/dev/null 2>&1 || true
fi

MSG="${TAG_PREFIX} ${NEW_TAG}"
echo "🏷️  创建 tag: ${NEW_TAG}"
echo "   消息: ${MSG}"
git tag -a "${NEW_TAG}" -m "${MSG}"

echo "🚀 推送 commit + tag 到 ${REMOTE}..."
# 推送当前分支（含版本 commit）与 tag
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "HEAD" ]; then
  git push "$REMOTE" "HEAD:refs/heads/${BRANCH}"
fi
git push "$REMOTE" "refs/tags/${NEW_TAG}"

echo "✅ 完成: ${NEW_TAG}"
echo "   本地开发默认版本 = ${APP_VERSION}（app.Version / wails.json）"
echo "   CI 发布仍会用 tag 经 ldflags 注入同一版本"
