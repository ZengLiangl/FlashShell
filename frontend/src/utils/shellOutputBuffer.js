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

/**
 * 每会话写队列：同一帧内合并 chunk，避免高速率 shell:data 连续打满 rAF/xterm.write。
 * @type {Map<string, {
 *   items: Array<{ type: string, content: string | Uint8Array }>,
 *   raf: number,
 *   flushing: boolean,
 *   pendingBytes: number,
 * }>}
 */
const writeQueues = new Map()

/** 写队列积压字节上限：超过则丢掉最旧的待写项（持久缓冲仍由 trimBuffer 管） */
const WRITE_QUEUE_PENDING_MAX_BYTES = 512 * 1024

/** 写队列积压条目上限（辅助背压） */
const WRITE_QUEUE_PENDING_MAX_ITEMS = 64

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

function writeBufferItem(writer, item) {
  const result = item.type === 'data'
    ? writer.writeData(item.content)
    : writer.writeLine(item.content)
  return Promise.resolve(result)
}

function concatUint8(chunks) {
  let total = 0
  for (const c of chunks) total += c.byteLength
  const out = new Uint8Array(total)
  let offset = 0
  for (const c of chunks) {
    out.set(c, offset)
    offset += c.byteLength
  }
  return out
}

function getWriteQueue(machineName) {
  let q = writeQueues.get(machineName)
  if (!q) {
    q = { items: [], raf: 0, flushing: false, pendingBytes: 0 }
    writeQueues.set(machineName, q)
  }
  return q
}

function clearWriteQueue(machineName) {
  const q = writeQueues.get(machineName)
  if (!q) return
  if (q.raf) {
    cancelAnimationFrame(q.raf)
    q.raf = 0
  }
  q.items = []
  q.pendingBytes = 0
  q.flushing = false
  writeQueues.delete(machineName)
}

/** 背压：丢弃写队列最旧项，优先合并后的大块仍会整块丢弃 */
function trimWriteQueue(q) {
  while (
    (q.pendingBytes > WRITE_QUEUE_PENDING_MAX_BYTES || q.items.length > WRITE_QUEUE_PENDING_MAX_ITEMS)
    && q.items.length > 0
  ) {
    const removed = q.items.shift()
    q.pendingBytes -= estimateSize(removed.type, removed.content)
    if (q.pendingBytes < 0) q.pendingBytes = 0
  }
}

/**
 * 将队列中连续 data 合并为单次 write，status line 穿插处打断。
 * @returns {Array<{ type: string, content: string | Uint8Array }>}
 */
function coalesceQueueItems(items) {
  if (!items.length) return []
  /** @type {Array<{ type: string, content: string | Uint8Array }>} */
  const out = []
  /** @type {Uint8Array[]} */
  let dataBatch = []
  const flushData = () => {
    if (!dataBatch.length) return
    const merged = dataBatch.length === 1 ? dataBatch[0] : concatUint8(dataBatch)
    dataBatch = []
    out.push({ type: 'data', content: merged })
  }
  for (const item of items) {
    if (item.type === 'data' && item.content instanceof Uint8Array) {
      dataBatch.push(item.content)
      continue
    }
    flushData()
    out.push(item)
  }
  flushData()
  return out
}

async function flushWriteQueue(machineName) {
  const q = writeQueues.get(machineName)
  if (!q) return
  q.raf = 0
  if (q.flushing) {
    // 上一帧还在写：本帧再排一次，避免丢后续 chunk
    q.raf = requestAnimationFrame(() => {
      flushWriteQueue(machineName).catch(() => {})
    })
    return
  }
  const writer = writers.get(machineName)
  if (!writer || !q.items.length) {
    q.items = []
    q.pendingBytes = 0
    return
  }

  const batch = q.items
  q.items = []
  q.pendingBytes = 0
  q.flushing = true
  try {
    const coalesced = coalesceQueueItems(batch)
    for (const item of coalesced) {
      // 写过程中 writer 可能被 hibernate/detach
      if (writers.get(machineName) !== writer) break
      await writeBufferItem(writer, item)
    }
  } catch {
    // ignore write errors
  } finally {
    q.flushing = false
    if (q.items.length && writers.get(machineName) === writer) {
      q.raf = requestAnimationFrame(() => {
        flushWriteQueue(machineName).catch(() => {})
      })
    }
  }
}

function enqueueWriterOutput(machineName, type, content) {
  const writer = writers.get(machineName)
  if (!writer) return
  const q = getWriteQueue(machineName)
  q.items.push({ type, content })
  q.pendingBytes += estimateSize(type, content)
  trimWriteQueue(q)
  if (!q.raf) {
    q.raf = requestAnimationFrame(() => {
      flushWriteQueue(machineName).catch(() => {})
    })
  }
}

/** 回放缓冲；合并连续 data 块以减少中间滚动闪烁 */
async function replayBufferItems(buf, writer, fromIndex = 0) {
  const start = Math.max(0, fromIndex)
  /** @type {Uint8Array[]} */
  let dataBatch = []
  const flushData = async () => {
    if (!dataBatch.length) return
    const merged = dataBatch.length === 1 ? dataBatch[0] : concatUint8(dataBatch)
    dataBatch = []
    await Promise.resolve(writer.writeData(merged))
  }
  for (let i = start; i < buf.items.length; i++) {
    const item = buf.items[i]
    if (item.type === 'data' && item.content instanceof Uint8Array) {
      dataBatch.push(item.content)
      continue
    }
    await flushData()
    await writeBufferItem(writer, item)
  }
  await flushData()
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
  // 有 writer 时入队合并写；标记已「交给终端侧」，避免 wake 时重复回放
  enqueueWriterOutput(machineName, type, normalized)
  buf.replayedCount = buf.items.length
}

export function clearShellOutput(machineName) {
  buffers.delete(machineName)
  clearWriteQueue(machineName)
  writers.get(machineName)?.clear()
}

export function removeShellOutput(machineName) {
  buffers.delete(machineName)
  clearWriteQueue(machineName)
  writers.delete(machineName)
  activeSessions.delete(machineName)
}

/** 丢弃缓冲但不清空终端画面（软断开后重连时避免回放重复输出） */
export function discardShellOutputBuffer(machineName) {
  buffers.delete(machineName)
  clearWriteQueue(machineName)
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

  clearWriteQueue(fromName)
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
    clearWriteQueue(fromName)
    writers.delete(fromName)
    activeSessions.delete(fromName)
    return
  }

  migrateShellOutput(fromName, toName)
}

/**
 * @param {object} [options]
 * @param {boolean} [options.replay=true]
 * @param {(phase: 'start' | 'end') => void} [options.aroundReplay] 回放开始/结束钩子（end 在全部 write 完成之后）
 * @returns {() => void} unregister
 */
export function registerShellWriter(machineName, writer, options = {}) {
  const { replay = true, aroundReplay } = options
  clearWriteQueue(machineName)
  writers.set(machineName, writer)
  const unregister = () => {
    if (writers.get(machineName) === writer) {
      clearWriteQueue(machineName)
      writers.delete(machineName)
    }
  }
  const buf = buffers.get(machineName)
  if (!buf || !replay) {
    return unregister
  }
  const from = buf.replayedCount || 0
  if (from >= buf.items.length) {
    return unregister
  }
  aroundReplay?.('start')
  replayBufferItems(buf, writer, from)
    .catch(() => {})
    .finally(() => {
      aroundReplay?.('end')
    })
  return unregister
}

/** 终端销毁后重置回放游标，下次挂载可完整回放仍在缓冲中的内容 */
export function resetShellWriterReplay(machineName) {
  const buf = buffers.get(machineName)
  if (buf) buf.replayedCount = 0
  clearWriteQueue(machineName)
}
