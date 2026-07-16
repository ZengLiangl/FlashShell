/** 从远程会话 ID 解析机器配置名（需传入已知配置名集合以避免 va-test-66 误判） */
export function remoteConfigName(sessionID, knownNames) {
  const id = String(sessionID || '').trim()
  if (!id) return id
  if (knownNames?.has?.(id)) return id

  if (knownNames) {
    for (const cfg of knownNames) {
      if (sessionBelongsToConfig(id, cfg)) return cfg
    }
  }

  const hash = id.match(/^(.+)#(\d+)$/)
  if (hash && parseInt(hash[2], 10) >= 2) {
    if (!knownNames || knownNames.has(hash[1])) return hash[1]
  }

  const dash = id.match(/^(.+)-(\d+)$/)
  if (dash && parseInt(dash[2], 10) >= 2) {
    if (!knownNames || knownNames.has(dash[1])) return dash[1]
  }

  return id
}

/** 会话 ID 是否属于某机器配置（web1 / web1-2、va-test-66 / va-test-66-2） */
export function sessionBelongsToConfig(sessionID, configName) {
  const id = String(sessionID || '').trim()
  const cfg = String(configName || '').trim()
  if (!id || !cfg) return false
  if (id === cfg) return true
  const m = id.match(new RegExp(`^${escapeRegExp(cfg)}-(\\d+)$`))
  return !!m && parseInt(m[1], 10) >= 2
}

function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function buildKnownMachineNames(machines) {
  const set = new Set()
  for (const m of machines || []) {
    if (m?.name) set.add(m.name)
  }
  return set
}
