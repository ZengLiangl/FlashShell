const MAX_BUFFER_BYTES = 2 * 1024 * 1024

/** @type {Map<string, { bytes: number, items: Array<{ type: string, content: string }>, replayedCount: number }>} */
const buffers = new Map()

/** @type {Map<string, { writeData: Function, writeLine: Function, clear: Function }>} */
const writers = new Map()

function getBuffer(machineName) {
  if (!buffers.has(machineName)) {
    buffers.set(machineName, { bytes: 0, items: [], replayedCount: 0 })
  }
  return buffers.get(machineName)
}

function estimateSize(type, content) {
  if (type === 'data') return content.length
  return (content?.length || 0) * 2
}

function trimBuffer(buf) {
  while (buf.bytes > MAX_BUFFER_BYTES && buf.items.length > 0) {
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
  if (!machineName || !content) return
  const buf = getBuffer(machineName)
  buf.items.push({ type, content })
  buf.bytes += estimateSize(type, content)
  trimBuffer(buf)

  const writer = writers.get(machineName)
  if (!writer) return
  if (type === 'data') writer.writeData(content)
  else writer.writeLine(content)
  buf.replayedCount = buf.items.length
}

export function clearShellOutput(machineName) {
  buffers.delete(machineName)
  writers.get(machineName)?.clear()
}

export function removeShellOutput(machineName) {
  buffers.delete(machineName)
  writers.delete(machineName)
}

/** 丢弃缓冲但不清空终端画面（软断开后重连时避免回放重复输出） */
export function discardShellOutputBuffer(machineName) {
  buffers.delete(machineName)
}

/** 将某会话的输出缓冲迁移到新会话 ID（连接占位 tab 替换为真实 ID） */
export function migrateShellOutput(fromName, toName) {
  if (!fromName || !toName || fromName === toName) return
  const buf = buffers.get(fromName)
  if (buf) {
    buffers.set(toName, buf)
    buffers.delete(fromName)
  }
  const writer = writers.get(fromName)
  if (writer) {
    writers.set(toName, writer)
    writers.delete(fromName)
  }
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
