/** 从远程会话 ID 解析机器配置名（需传入已知配置名集合以避免 va-test-66 误判） */
export function remoteConfigName(sessionID, knownNames) {
  const id = String(sessionID || '').trim()
  if (!id) return id
  if (knownNames?.has?.(id)) return id

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

export function buildKnownMachineNames(machines) {
  const set = new Set()
  for (const m of machines || []) {
    if (m?.name) set.add(m.name)
  }
  return set
}
