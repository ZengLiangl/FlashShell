export const DEFAULT_MACHINE_GROUP = '默认分组'

export function getMachineGroup(machine) {
  const group = machine?.group?.trim()
  return group || DEFAULT_MACHINE_GROUP
}

/** 展示用地址：user@host（默认端口 22 时省略 :port） */
export function formatMachineAddr(machine) {
  const user = String(machine?.user || '').trim()
  const host = String(machine?.host || machine?.ip || '').trim()
  const port = Number(machine?.port) || 22
  if (!host) return machine?.key_file || '-'
  const auth = user ? `${user}@${host}` : host
  return port === 22 ? auth : `${auth}:${port}`
}

/** 机器列表关键词匹配：名称、用户、IP/主机、分组、密钥路径 */
export function machineMatchesKeyword(machine, keyword) {
  const kw = String(keyword || '').trim().toLowerCase()
  if (!kw) return true
  const host = String(machine?.host || machine?.ip || '').toLowerCase()
  return (
    String(machine?.name || '').toLowerCase().includes(kw) ||
    String(machine?.user || '').toLowerCase().includes(kw) ||
    host.includes(kw) ||
    String(machine?.group || '').toLowerCase().includes(kw) ||
    String(machine?.key_file || '').toLowerCase().includes(kw) ||
    formatMachineAddr(machine).toLowerCase().includes(kw)
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

/**
 * 树形展示用：自定义分组可折叠；默认分组机器平铺在根级（不包一层「默认分组」）。
 * 顺序：自定义分组 → 默认分组机器。
 */
export function splitMachineTree(machines) {
  const groups = groupMachines(machines)
  const customGroups = []
  let defaultMachines = []
  for (const g of groups) {
    if (g.name === DEFAULT_MACHINE_GROUP) {
      defaultMachines = g.machines
    } else {
      customGroups.push(g)
    }
  }
  return { customGroups, defaultMachines }
}

export function isMachineConnected(machineName, sessions) {
  return (sessions || []).some(
    (s) =>
      s.connected &&
      (s.machineName === machineName || s.configName === machineName),
  )
}

/** 某机器配置是否正在连接（含占位 tab） */
export function isMachineConnecting(configName, workspaceSessions = []) {
  if (!configName) return false
  return (workspaceSessions || []).some(
    (s) =>
      s.connecting &&
      (s.configName === configName || s.machineName === configName),
  )
}

/** 某机器配置当前活动会话数 */
export function countMachineSessions(configName, sessions) {
  return (sessions || []).filter(
    (s) => s.connected && (s.configName === configName || s.machineName === configName),
  ).length
}
