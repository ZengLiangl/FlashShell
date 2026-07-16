/** 检查更新结果的进程内缓存（不落盘），默认 30 分钟 */
const TTL_MS = 30 * 60 * 1000

let cached = null // { result, fetchedAt }

export function isUsableUpdateResult(result) {
  if (!result || result.error) return false
  return !!String(result.latestVersion || '').trim()
}

export function getCachedUpdateCheck() {
  if (!cached) return null
  if (Date.now() - cached.fetchedAt > TTL_MS) {
    cached = null
    return null
  }
  return cached.result
}

export function setCachedUpdateCheck(result) {
  if (!isUsableUpdateResult(result)) return
  cached = {
    result,
    fetchedAt: Date.now(),
  }
}

export function clearCachedUpdateCheck() {
  cached = null
}
