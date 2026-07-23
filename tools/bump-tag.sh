#!/usr/bin/env bash
# bump-tag.sh — 打 annotated tag 并推送；版本号提交放在 tag 之后
#
# 用法:
#   ./tools/bump-tag.sh                         # 拉取最新 v* tag，patch +1
#   ./tools/bump-tag.sh v1.0.3                  # 直接使用指定版本
#   ./tools/bump-tag.sh --base v1.0.1           # Release = 历史版本说明 + 本次提交
#   ./tools/bump-tag.sh v1.0.3 --base v1.0.1
#   RELEASE_BASE=v1.0.1 ./tools/bump-tag.sh
#
# 流程（重要）:
#   1. 解析 NEW_TAG（如 v1.0.4）与可选 RELEASE_BASE
#   2. 若指定 --base：写入 .github/release-base 并先 commit（CI 可靠读取，不依赖 tag 注解）
#   3. 在「当前 HEAD」上打 tag 并先推送 tag
#      → Release / changelog 不含 "bump version" / "set release-base" 提交
#      → tag 注解仍写入 release-base: 作为备份
#   4. 再写回 app/version.go、wails.json，清除 .github/release-base，并 commit
#   5. 推送版本 commit 到当前分支
#
# 说明:
#   CI 发布用 tag + ldflags 注入版本，不依赖 tag 指向的 commit 里是否已改 version.go。
#   bump commit 仅同步本地/后续开发读到的默认版本号。
#   Release body：未指定 base 时仅本次提交；指定后按类型合并「该历史版本 GitHub Release body + 本次提交」（无「本次更新」分段）。
#   CI 优先读仓库内 .github/release-base（比 annotated tag 注解更可靠）。
#
# 环境变量:
#   REMOTE=origin  TAG_PREFIX=FlashDock
#   RELEASE_BASE=v1.0.1  等同 --base（历史版本 tag）
#   SKIP_COMMIT=1  只打 tag（不写版本文件、不提交）
#   SKIP_BUMP=1    打完 tag 后不写/不提交版本文件
#
# 重打同名版本时会删除本地/远程 tag；若已安装 gh，还会删除同名 GitHub Release，
# 避免 CI 更新旧 Release 时沿用错误 body。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REMOTE="${REMOTE:-origin}"
TAG_PREFIX="${TAG_PREFIX:-FlashDock}"
RELEASE_BASE="${RELEASE_BASE:-}"
SKIP_COMMIT="${SKIP_COMMIT:-0}"
SKIP_BUMP="${SKIP_BUMP:-0}"

VERSION_GO="app/version.go"
WAILS_JSON="wails.json"
RELEASE_BASE_FILE=".github/release-base"

EXPLICIT_TAG=""
while [ $# -gt 0 ]; do
  case "$1" in
    --base)
      if [ $# -lt 2 ]; then
        echo "❌ --base 需要版本参数，例如 --base v1.0.1" >&2
        exit 1
      fi
      RELEASE_BASE="$2"
      shift 2
      ;;
    --base=*)
      RELEASE_BASE="${1#--base=}"
      shift
      ;;
    -h|--help)
      sed -n '2,28p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    -*)
      echo "❌ 未知参数: $1（支持 --base <tag>）" >&2
      exit 1
      ;;
    *)
      if [ -n "$EXPLICIT_TAG" ]; then
        echo "❌ 多余参数: $1" >&2
        exit 1
      fi
      EXPLICIT_TAG="$1"
      shift
      ;;
  esac
done

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

if [ -n "$RELEASE_BASE" ]; then
  RELEASE_BASE="$(normalize_tag "$RELEASE_BASE")"
  validate_tag "$RELEASE_BASE"
  if [ "$RELEASE_BASE" = "$NEW_TAG" ]; then
    echo "❌ --base 不能与新版本相同: ${RELEASE_BASE}" >&2
    exit 1
  fi
  echo "📎 Release body 将基于历史版本: ${RELEASE_BASE} + 本次提交"
fi

APP_VERSION="${NEW_TAG#v}"
APP_VERSION="${APP_VERSION%%[-+]*}"
export APP_VERSION

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
HEAD_SHA="$(git rev-parse HEAD)"

if [ "$SKIP_COMMIT" = "1" ]; then
  echo "⏭️  SKIP_COMMIT=1：仅解析版本 ${NEW_TAG}，不打 tag / 不提交"
  sync_version_files "$NEW_TAG"
  exit 0
fi

# 打 tag 前若工作区脏，先警告（避免误以为版本文件会进 tag）
if [ -n "$(git status --porcelain)" ]; then
  echo "⚠️  工作区有未提交变更；tag 将指向当前 HEAD=${HEAD_SHA:0:8}"
  echo "   bump version 提交会在打 tag 之后进行"
fi

if git rev-parse -q --verify "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "🗑️  删除本地已有 tag ${NEW_TAG}"
  git tag -d "${NEW_TAG}" >/dev/null 2>&1 || true
fi

if git ls-remote --tags --exit-code "$REMOTE" "refs/tags/${NEW_TAG}" >/dev/null 2>&1; then
  echo "🗑️  删除远程已有 tag ${NEW_TAG}"
  git push "$REMOTE" ":refs/tags/${NEW_TAG}" >/dev/null 2>&1 || true
fi

# 重打同名 tag 时，旧 Release 可能残留；先删掉让 CI 重新创建（含正确 body）
if command -v gh >/dev/null 2>&1; then
  if gh release view "${NEW_TAG}" >/dev/null 2>&1; then
    echo "🗑️  删除已有 GitHub Release ${NEW_TAG}（避免沿用旧 body）"
    gh release delete "${NEW_TAG}" --yes 2>/dev/null || true
  fi
else
  echo "ℹ️  未安装 gh CLI：若远端已有 ${NEW_TAG} Release，请手动删除后再发版"
fi

# —— 打 tag 之前写入 release-base 文件并提交，保证 CI checkout 一定能读到 ——
if [ -n "$RELEASE_BASE" ]; then
  mkdir -p "$(dirname "$RELEASE_BASE_FILE")"
  printf '%s\n' "$RELEASE_BASE" > "$RELEASE_BASE_FILE"
  git add -- "$RELEASE_BASE_FILE"
  if ! git diff --cached --quiet -- "$RELEASE_BASE_FILE"; then
    git commit -m "chore: set release-base to ${RELEASE_BASE}"
    echo "✅ 已提交 ${RELEASE_BASE_FILE}=${RELEASE_BASE}（在 tag 之前，供 CI 读取）"
  else
    echo "ℹ️  ${RELEASE_BASE_FILE} 已是 ${RELEASE_BASE}"
  fi
  # 先推送该提交，避免只推 tag 时远端缺少此 commit（部分托管会出现）
  if [ "$BRANCH" != "HEAD" ]; then
    git push "$REMOTE" "HEAD:refs/heads/${BRANCH}"
  fi
  HEAD_SHA="$(git rev-parse HEAD)"
fi

MSG="${TAG_PREFIX} ${NEW_TAG}"
if [ -n "$RELEASE_BASE" ]; then
  # 备份：CI 也可从 annotated tag 正文读取（不可靠时以文件为准）
  MSG="${MSG}"$'\n\n'"release-base: ${RELEASE_BASE}"
fi
echo "🏷️  在当前 HEAD 创建 tag: ${NEW_TAG}"
echo "   指向: ${HEAD_SHA}"
echo "   消息: ${MSG}"
git tag -a "${NEW_TAG}" -m "${MSG}" "${HEAD_SHA}"

echo "🚀 先推送 tag 到 ${REMOTE}（触发 Release，不含 bump commit）..."
git push "$REMOTE" "refs/tags/${NEW_TAG}"

if [ "$SKIP_BUMP" = "1" ]; then
  echo "⏭️  SKIP_BUMP=1，跳过版本文件提交"
  echo "✅ 完成: ${NEW_TAG}"
  exit 0
fi

# —— tag 之后再 bump，保证 release body 看不到这条提交 ——
sync_version_files "$NEW_TAG"

# 清除 release-base，避免下一版本误用
if [ -f "$RELEASE_BASE_FILE" ] || git ls-files --error-unmatch -- "$RELEASE_BASE_FILE" >/dev/null 2>&1; then
  git rm -f --ignore-unmatch -- "$RELEASE_BASE_FILE" >/dev/null 2>&1 || rm -f -- "$RELEASE_BASE_FILE"
fi

git add -- "$VERSION_GO" "$WAILS_JSON" 2>/dev/null || true
if git ls-files --deleted -- "$RELEASE_BASE_FILE" 2>/dev/null | grep -q .; then
  git add -u -- "$RELEASE_BASE_FILE"
fi

if ! git diff --cached --quiet; then
  git commit -m "chore: bump version to ${APP_VERSION}"
  echo "✅ 已提交版本号变更: ${APP_VERSION}（在 tag 之后；已清除 release-base）"
else
  echo "ℹ️  版本文件已是 ${APP_VERSION}，无需额外 commit"
fi

if [ "$BRANCH" != "HEAD" ]; then
  echo "🚀 推送分支 ${BRANCH}（含 bump commit）..."
  git push "$REMOTE" "HEAD:refs/heads/${BRANCH}"
fi

echo "✅ 完成: ${NEW_TAG}"
echo "   tag 指向发布代码（无 bump commit）"
if [ -n "$RELEASE_BASE" ]; then
  echo "   Release body = ${RELEASE_BASE} 说明 + 本次提交（CI 读 ${RELEASE_BASE_FILE}）"
fi
echo "   本地默认版本 = ${APP_VERSION}（app.Version / wails.json）"
echo "   CI 发布仍会用 tag 经 ldflags 注入同一版本"
