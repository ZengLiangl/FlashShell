import {
  SHELL_OUTPUT_BUFFER_ACTIVE_MAX,
  SHELL_OUTPUT_BUFFER_BACKGROUND_MAX,
} from '../constants/shellMemory'

/** @type {Map<string, { bytes: number, items: Array<{ type: string, content: string | Uint8Array }>, replayedCount: number }>} */
const buffers = new Map()

/** @type {Map<string, { writeData: Function, writeLine: Function, clear: Function }>} */
const writers = new Map()

/** @type {Set<string>} */
const activeSessions = new Set()

function getMaxBufferBytes(machineName) {
  return activeSessions.has(machineName)
    ? SHELL_OUTPUT_BUFFER_ACTIVE_MAX
    : SHELL_OUTPUT_BUFFER_BACKGROUND_MAX
}

export function setShellOutputSessionActive(machineName, active) {
  if (!machineName) return
  if (active) activeSessions.add(machineName)
  else activeSessions.delete(machineName)
  trimBuffer(getBuffer(machineName), machineName)
}

function getBuffer(machineName) {
  if (!buffers.has(machineName)) {
    buffers.set(machineName, { bytes: 0, items: [], replayedCount: 0 })
  }
  return buffers.get(machineName)
}

/** base64 字符串 → Uint8Array（Wails 事件 JSON 传输仍为 base64，此处只解码一次） */
export function decodeShellOutputData(content) {
  if (content instanceof Uint8Array) return content
  if (typeof content !== 'string' || !content) return new Uint8Array(0)
  const binary = atob(content)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

function normalizeContent(type, content) {
  if (type === 'data') return decodeShellOutputData(content)
  return content
}

function estimateSize(type, content) {
  if (type === 'data') {
    return content instanceof Uint8Array ? content.byteLength : (content?.length || 0)
  }
  return (content?.length || 0) * 2
}

function trimBuffer(buf, machineName) {
  const maxBytes = getMaxBufferBytes(machineName)
  while (buf.bytes > maxBytes && buf.items.length > 0) {
    const removed = buf.items.shift()
    buf.bytes -= estimateSize(removed.type, removed.content)
    if (buf.replayedCount > 0) buf.replayedCount -= 1
  }
}

function replayBufferItems(buf, writer, fromIndex = 0) {
  const start = Math.max(0, fromIndex)
  for (let i = start; i < buf.items.length; i++) {
    const item = buf.items[i]
    if (item.type === 'data') writer.writeData(item.content)
    else writer.writeLine(item.content)
  }
  buf.replayedCount = buf.items.length
}

export function pushShellOutput(machineName, type, content) {
  if (!machineName || content == null || content === '') return
  const normalized = normalizeContent(type, content)
  if (type === 'data' && normalized.byteLength === 0) return

  const buf = getBuffer(machineName)
  buf.items.push({ type, content: normalized })
  buf.bytes += estimateSize(type, normalized)
  trimBuffer(buf, machineName)

  const writer = writers.get(machineName)
  if (!writer) return
  if (type === 'data') writer.writeData(normalized)
  else writer.writeLine(normalized)
  buf.replayedCount = buf.items.length
}

export function clearShellOutput(machineName) {
  buffers.delete(machineName)
  writers.get(machineName)?.clear()
}

export function removeShellOutput(machineName) {
  buffers.delete(machineName)
  writers.delete(machineName)
  activeSessions.delete(machineName)
}

/** 丢弃缓冲但不清空终端画面（软断开后重连时避免回放重复输出） */
export function discardShellOutputBuffer(machineName) {
  buffers.delete(machineName)
}

/** 将某会话的输出缓冲迁移到新会话 ID（连接占位 tab 替换为真实 ID） */
export function migrateShellOutput(fromName, toName) {
  if (!fromName || !toName || fromName === toName) return
  const fromBuf = buffers.get(fromName)
  const toBuf = buffers.get(toName)

  if (fromBuf && toBuf && fromBuf !== toBuf) {
    for (const item of fromBuf.items) {
      toBuf.items.push(item)
      toBuf.bytes += estimateSize(item.type, item.content)
    }
    trimBuffer(toBuf, toName)
    buffers.delete(fromName)
  } else if (fromBuf && !toBuf) {
    buffers.set(toName, fromBuf)
    buffers.delete(fromName)
  } else {
    buffers.delete(fromName)
  }

  // writer 绑定具体 xterm 实例，不随 ID 迁移；新终端挂载时会重新 register
  writers.delete(fromName)

  if (activeSessions.has(fromName)) {
    activeSessions.add(toName)
    activeSessions.delete(fromName)
  }
}

/** 连接完成时合并输出缓冲：目标已有 PTY 输出时只追加 pending 内容，不覆盖 */
export function finalizeShellOutputMigration(fromName, toName) {
  if (!fromName || !toName || fromName === toName) return
  const fromBuf = buffers.get(fromName)
  const toBuf = buffers.get(toName)
  const fromLen = fromBuf?.items?.length || 0
  const toLen = toBuf?.items?.length || 0

  if (toLen > 0) {
    if (fromLen > 0 && fromBuf !== toBuf) {
      for (const item of fromBuf.items) {
        toBuf.items.push(item)
        toBuf.bytes += estimateSize(item.type, item.content)
      }
      trimBuffer(toBuf, toName)
    }
    buffers.delete(fromName)
    writers.delete(fromName)
    activeSessions.delete(fromName)
    return
  }

  migrateShellOutput(fromName, toName)
}

export function registerShellWriter(machineName, writer, options = {}) {
  const { replay = true } = options
  writers.set(machineName, writer)
  const buf = buffers.get(machineName)
  if (!buf || !replay) {
    return () => writers.delete(machineName)
  }
  replayBufferItems(buf, writer, buf.replayedCount || 0)
  return () => writers.delete(machineName)
}

/** 终端销毁后重置回放游标，下次挂载可完整回放仍在缓冲中的内容 */
export function resetShellWriterReplay(machineName) {
  const buf = buffers.get(machineName)
  if (buf) buf.replayedCount = 0
}
