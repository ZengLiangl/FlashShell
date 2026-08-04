/**
 * 解析快速连接地址：user@host、user@host:port、host:port、hostname
 * @returns {{ user: string, host: string, port: number } | null}
 */
export function parseQuickConnectTarget(input) {
  const raw = String(input || '').trim()
  if (!raw || /\s/.test(raw)) return null
  // 排除纯机器名风格（无 . : @）——仍允许 IP / FQDN / user@
  let user = ''
  let rest = raw
  const at = raw.lastIndexOf('@')
  if (at >= 0) {
    user = raw.slice(0, at)
    rest = raw.slice(at + 1)
    if (!user || !rest) return null
  }
  let host = rest
  let port = 0
  if (rest.startsWith('[') && rest.includes(']:')) {
    const end = rest.indexOf(']:')
    host = rest.slice(1, end)
    port = Number(rest.slice(end + 2)) || 0
  } else {
    const colon = rest.lastIndexOf(':')
    if (colon > 0 && /^\d+$/.test(rest.slice(colon + 1))) {
      host = rest.slice(0, colon)
      port = Number(rest.slice(colon + 1)) || 0
    }
  }
  if (!host) return null
  const looksAddr =
    at >= 0 ||
    host.includes('.') ||
    /^\d{1,3}(\.\d{1,3}){3}$/.test(host) ||
    port > 0
  if (!looksAddr) return null
  return { user, host, port }
}

export function findMachineForQuickConnect(machines, target) {
  if (!target?.host) return null
  const list = machines || []
  const hostEq = (m) => {
    const h = String(m?.host || m?.list_host || '').toLowerCase()
    return h === String(target.host).toLowerCase()
  }
  const userEq = (m) => {
    if (!target.user) return true
    const u = String(m?.user || m?.list_user || '')
    return u === target.user
  }
  const portEq = (m) => {
    if (!target.port) return true
    const p = Number(m?.port || m?.list_port || 0)
    return p === target.port || (!p && target.port === 22)
  }
  return list.find((m) => hostEq(m) && userEq(m) && portEq(m)) || null
}
