const HOST_KEY = 'flashdock.sftp.bookmarks.host.v1'
const GLOBAL_KEY = 'flashdock.sftp.bookmarks.global.v1'
const HISTORY_KEY = 'flashdock.sftp.pathHistory.v1'
const FOLLOW_KEY = 'flashdock.sftp.followCwd.v1'

function safeParse(raw, fallback) {
  try {
    const v = JSON.parse(raw || '')
    return v ?? fallback
  } catch {
    return fallback
  }
}

function readMap(key) {
  return safeParse(localStorage.getItem(key), {}) || {}
}

function writeMap(key, map) {
  localStorage.setItem(key, JSON.stringify(map || {}))
}

export function createSftpBookmark(path, { global = false } = {}) {
  const p = String(path || '').trim() || '/'
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    path: p,
    label: p,
    global: !!global,
  }
}

export function loadGlobalBookmarks() {
  const list = safeParse(localStorage.getItem(GLOBAL_KEY), [])
  return Array.isArray(list) ? list : []
}

export function saveGlobalBookmarks(list) {
  localStorage.setItem(GLOBAL_KEY, JSON.stringify(Array.isArray(list) ? list : []))
}

export function loadHostBookmarks(hostKey) {
  const key = String(hostKey || '').trim()
  if (!key) return []
  const map = readMap(HOST_KEY)
  const list = map[key]
  return Array.isArray(list) ? list : []
}

export function saveHostBookmarks(hostKey, list) {
  const key = String(hostKey || '').trim()
  if (!key) return
  const map = readMap(HOST_KEY)
  map[key] = Array.isArray(list) ? list : []
  writeMap(HOST_KEY, map)
}

export function mergedBookmarks(hostKey) {
  const global = loadGlobalBookmarks().map((b) => ({ ...b, global: true }))
  const host = loadHostBookmarks(hostKey).map((b) => ({ ...b, global: false }))
  const seen = new Set()
  const out = []
  for (const b of [...global, ...host]) {
    const p = String(b?.path || '')
    if (!p || seen.has(p)) continue
    seen.add(p)
    out.push(b)
  }
  return out
}

export function isPathBookmarked(hostKey, path) {
  const p = String(path || '')
  return mergedBookmarks(hostKey).some((b) => b.path === p)
}

export function toggleHostBookmark(hostKey, path) {
  const p = String(path || '').trim() || '/'
  const list = loadHostBookmarks(hostKey)
  const idx = list.findIndex((b) => b.path === p)
  if (idx >= 0) {
    list.splice(idx, 1)
    saveHostBookmarks(hostKey, list)
    return false
  }
  list.push(createSftpBookmark(p))
  saveHostBookmarks(hostKey, list)
  return true
}

export function addGlobalBookmark(path) {
  const p = String(path || '').trim() || '/'
  const list = loadGlobalBookmarks()
  if (list.some((b) => b.path === p)) return
  list.push(createSftpBookmark(p, { global: true }))
  saveGlobalBookmarks(list)
}

export function removeBookmark(hostKey, id) {
  const g = loadGlobalBookmarks()
  const nextG = g.filter((b) => b.id !== id)
  if (nextG.length !== g.length) {
    saveGlobalBookmarks(nextG)
    return
  }
  const h = loadHostBookmarks(hostKey)
  saveHostBookmarks(hostKey, h.filter((b) => b.id !== id))
}

export function loadPathHistory(hostKey) {
  const key = String(hostKey || '').trim()
  if (!key) return []
  const map = readMap(HISTORY_KEY)
  const list = map[key]
  return Array.isArray(list) ? list : []
}

export function pushPathHistory(hostKey, path, limit = 20) {
  const key = String(hostKey || '').trim()
  const p = String(path || '').trim()
  if (!key || !p) return
  const map = readMap(HISTORY_KEY)
  const prev = Array.isArray(map[key]) ? map[key] : []
  const next = [p, ...prev.filter((x) => x !== p)].slice(0, limit)
  map[key] = next
  writeMap(HISTORY_KEY, map)
}

export function loadFollowCwd(hostKey) {
  const key = String(hostKey || '').trim() || '_'
  const map = readMap(FOLLOW_KEY)
  if (Object.prototype.hasOwnProperty.call(map, key)) return !!map[key]
  return true
}

export function saveFollowCwd(hostKey, follow) {
  const key = String(hostKey || '').trim() || '_'
  const map = readMap(FOLLOW_KEY)
  map[key] = !!follow
  writeMap(FOLLOW_KEY, map)
}

export function suggestPaths({ hostKey, draft, folderEntries = [] }) {
  const q = String(draft || '').trim()
  const out = []
  const seen = new Set()
  const push = (path, type) => {
    const p = String(path || '')
    if (!p || seen.has(p)) return
    if (q && !p.toLowerCase().includes(q.toLowerCase()) && !p.startsWith(q)) return
    seen.add(p)
    out.push({ path: p, type })
  }
  for (const b of mergedBookmarks(hostKey)) push(b.path, 'bookmark')
  for (const h of loadPathHistory(hostKey)) push(h, 'history')
  for (const e of folderEntries || []) {
    if (e?.isDir && e.path) push(e.path, 'folder')
  }
  return out.slice(0, 12)
}
