import { CreateMachine, GetMachineSensitiveData } from '../../wailsjs/go/app/App'
import { DEFAULT_MACHINE_GROUP } from './machineGroups'

function normalizeGroup(g) {
  const s = String(g || '').trim()
  if (!s || s === DEFAULT_MACHINE_GROUP) return ''
  return s
}

/** 在名称后追加 -copy，若重名则递增 -copy2、-copy3… */
export function buildCopyMachineName(baseName, existingNames) {
  const names = new Set((existingNames || []).map((n) => String(n || '').trim()).filter(Boolean))
  const root = String(baseName || '').trim() || 'machine'
  let candidate = `${root}-copy`
  if (!names.has(candidate)) return candidate
  let i = 2
  while (names.has(`${root}-copy${i}`)) i += 1
  return `${root}-copy${i}`
}

/** 复制一条机器配置（含敏感字段与隧道） */
export async function copyMachineRecord(source, existingMachines) {
  if (!source?.id) throw new Error('无效的机器配置')

  const existingNames = (existingMachines || []).map((m) => m.name)
  const copyName = buildCopyMachineName(source.name, existingNames)

  const sensitiveData = await GetMachineSensitiveData(source.id)
  const machineData = {
    name: copyName,
    group: normalizeGroup(source.group),
    key_file: source.key_file || '',
    tags: Array.isArray(source.tags) ? [...source.tags] : [],
    notes: source.notes || '',
    icon: source.icon || '',
    proxyJump: source.proxyJump || '',
    jumpChain: Array.isArray(source.jumpChain) ? [...source.jumpChain] : [],
    legacyAlgorithms: !!source.legacyAlgorithms,
    skipEcdsaHostKey: !!source.skipEcdsaHostKey,
    sftpEncoding: source.sftpEncoding || 'auto',
    sftpFileProtocol: source.sftpFileProtocol || 'auto',
    sftpSudo: !!source.sftpSudo,
    startupCommand: source.startupCommand || '',
    agentForwarding: !!source.agentForwarding,
    tunnels: (source.tunnels || [])
      .filter((t) => t.localPort > 0)
      .map((t) => ({
        enabled: t.enabled !== false,
        name: t.name || '',
        type: t.type || 'local',
        localHost: t.localHost || '127.0.0.1',
        localPort: t.localPort || 0,
        remoteHost: t.remoteHost || '127.0.0.1',
        remotePort: t.remotePort || 0,
      })),
  }

  await CreateMachine(machineData, sensitiveData || {})
  return copyName
}
