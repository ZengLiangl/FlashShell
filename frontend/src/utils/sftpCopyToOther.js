/**
 * 跨会话「复制到另一侧」目标解析
 */

/**
 * @typedef {{ sessionId: string, destDir: string, label: string }} CopyToOtherTarget
 */

/**
 * 解析可复制到的目标会话列表。
 * 优先：当前分屏中的其他远程会话；否则：全部其他远程会话。
 *
 * @param {{
 *   sourceSessionId: string,
 *   splitSessionIds?: string[],
 *   sessions?: Array<{ machineName?: string, kind?: string, tabLabel?: string, configName?: string, connected?: boolean }>,
 *   cwdBySession?: Record<string, string>,
 *   isLocalSession?: (id: string) => boolean,
 * }} opts
 * @returns {CopyToOtherTarget[]}
 */
export function resolveCopyToOtherTargets(opts) {
  const sourceSessionId = String(opts?.sourceSessionId || '').trim()
  if (!sourceSessionId) return []

  const isLocal = typeof opts?.isLocalSession === 'function'
    ? opts.isLocalSession
    : (id) => {
        const n = String(id || '')
        return n === 'local' || n.startsWith('local-')
      }

  if (isLocal(sourceSessionId)) return []

  const sessions = Array.isArray(opts?.sessions) ? opts.sessions : []
  const cwdBySession = opts?.cwdBySession || {}
  const splitIds = (opts?.splitSessionIds || []).map((x) => String(x || '').trim()).filter(Boolean)

  const byId = new Map()
  for (const s of sessions) {
    const id = String(s?.machineName || '').trim()
    if (!id) continue
    byId.set(id, s)
  }

  /** @param {string[]} ids */
  const buildFromIds = (ids) => {
    const out = []
    for (const id of ids) {
      if (!id || id === sourceSessionId) continue
      if (isLocal(id)) continue
      const s = byId.get(id)
      if (!s) continue
      if (s.kind === 'local') continue
      if (s.connected === false) continue
      const destDir = normalizeDestDir(cwdBySession[id])
      if (!destDir) continue
      const label = String(s.tabLabel || s.configName || id).trim() || id
      out.push({
        sessionId: id,
        destDir,
        label: `${label} → ${destDir}`,
      })
    }
    return out
  }

  if (splitIds.includes(sourceSessionId) && splitIds.length >= 2) {
    const peers = buildFromIds(splitIds)
    if (peers.length) return peers
  }

  return buildFromIds(sessions.map((s) => String(s?.machineName || '').trim()).filter(Boolean))
}

/** @param {unknown} dir */
export function normalizeDestDir(dir) {
  let s = String(dir || '').trim()
  if (!s.startsWith('/')) return ''
  if (s.length > 1) s = s.replace(/\/+$/, '')
  return s
}

/**
 * @param {CopyToOtherTarget[]} targets
 * @returns {boolean}
 */
export function canCopyToOtherSide(targets) {
  return Array.isArray(targets) && targets.length > 0
}
