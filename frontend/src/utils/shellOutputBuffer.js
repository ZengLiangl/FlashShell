const MAX_BUFFER_BYTES = 2 * 1024 * 1024

/** @type {Map<string, { bytes: number, items: Array<{ type: string, content: string }> }>} */
const buffers = new Map()

/** @type {Map<string, { writeData: Function, writeLine: Function, clear: Function }>} */
const writers = new Map()

function getBuffer(machineName) {
  if (!buffers.has(machineName)) {
    buffers.set(machineName, { bytes: 0, items: [] })
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
  }
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

export function registerShellWriter(machineName, writer) {
  writers.set(machineName, writer)
  const buf = buffers.get(machineName)
  if (!buf) return () => writers.delete(machineName)
  for (const item of buf.items) {
    if (item.type === 'data') writer.writeData(item.content)
    else writer.writeLine(item.content)
  }
  return () => writers.delete(machineName)
}
