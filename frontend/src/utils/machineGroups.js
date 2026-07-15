export const DEFAULT_MACHINE_GROUP = '默认分组'

export function getMachineGroup(machine) {
  const group = machine?.group?.trim()
  return group || DEFAULT_MACHINE_GROUP
}

/** 机器列表关键词匹配：名称、IP/主机、分组、密钥路径 */
export function machineMatchesKeyword(machine, keyword) {
  const kw = String(keyword || '').trim().toLowerCase()
  if (!kw) return true
  const host = String(machine?.host || machine?.ip || '').toLowerCase()
  return (
    String(machine?.name || '').toLowerCase().includes(kw) ||
    host.includes(kw) ||
    String(machine?.group || '').toLowerCase().includes(kw) ||
    String(machine?.key_file || '').toLowerCase().includes(kw)
  )
}

/** 机器名称首字母排序：a-z，再 0-9，其它靠后；同前缀按完整名 localeCompare */
export function compareMachineName(a, b) {
  const na = String(a?.name || a || '')
  const nb = String(b?.name || b || '')
  const ka = sortKeyForName(na)
  const kb = sortKeyForName(nb)
  if (ka.bucket !== kb.bucket) return ka.bucket - kb.bucket
  if (ka.head !== kb.head) return ka.head < kb.head ? -1 : 1
  return na.localeCompare(nb, undefined, { sensitivity: 'base', numeric: true })
}

function sortKeyForName(name) {
  const trimmed = String(name || '').trim()
  const ch = trimmed.charAt(0).toLowerCase()
  if (ch >= 'a' && ch <= 'z') {
    return { bucket: 0, head: ch }
  }
  if (ch >= '0' && ch <= '9') {
    return { bucket: 1, head: ch }
  }
  return { bucket: 2, head: ch || '\uffff' }
}

export function sortMachinesByName(machines) {
  return (machines || []).slice().sort(compareMachineName)
}

export function groupMachines(machines) {
  const groups = {}
  for (const machine of sortMachinesByName(machines)) {
    const name = getMachineGroup(machine)
    if (!groups[name]) groups[name] = []
    groups[name].push(machine)
  }
  const sorted = Object.keys(groups).sort((a, b) => {
    if (a === DEFAULT_MACHINE_GROUP) return -1
    if (b === DEFAULT_MACHINE_GROUP) return 1
    return a.localeCompare(b, 'zh-CN')
  })
  return sorted.map((name) => ({ name, machines: groups[name] }))
}

export function isMachineConnected(machineName, sessions) {
  return (sessions || []).some((s) => s.machineName === machineName && s.connected)
}
