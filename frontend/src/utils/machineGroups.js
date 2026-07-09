export const DEFAULT_MACHINE_GROUP = '默认分组'

export function getMachineGroup(machine) {
  const group = machine?.group?.trim()
  return group || DEFAULT_MACHINE_GROUP
}

export function groupMachines(machines) {
  const groups = {}
  for (const machine of machines || []) {
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
