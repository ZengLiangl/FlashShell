/** 从错误信息解析未知主机密钥 */
export function parseHostKeyError(errOrMsg) {
  const msg = String(errOrMsg?.message || errOrMsg || '')
  const m = msg.match(/未知主机密钥\s+([^（]+)（指纹\s+(SHA256:[A-Za-z0-9+/=]+)）/)
  if (m) {
    const hostPort = m[1].trim()
    const [host, portStr] = hostPort.includes(':')
      ? [hostPort.slice(0, hostPort.lastIndexOf(':')), hostPort.slice(hostPort.lastIndexOf(':') + 1)]
      : [hostPort, '22']
    return {
      host: host.trim(),
      port: parseInt(portStr, 10) || 22,
      fingerprint: m[2],
      raw: msg,
    }
  }
  return null
}

export function isHostKeyError(errOrMsg) {
  return !!parseHostKeyError(errOrMsg)
}
