/**
 * 读取 xterm 选区文本；合并软换行，避免 less 等分页器把视觉换行拷成硬换行。
 * tail/grep 依赖终端 soft-wrap（isWrapped）；less 常把续行写成满宽硬换行。
 */
export function getTerminalSelectionText(term) {
  if (!term) return ''
  const pos = term.getSelectionPosition?.()
  if (!pos) return term.getSelection?.() || ''

  const buf = term.buffer?.active
  if (!buf || typeof buf.getLine !== 'function') {
    return term.getSelection?.() || ''
  }

  const cols = term.cols || 80
  const startY = pos.start.y
  const endY = pos.end.y
  let out = ''

  for (let y = startY; y <= endY; y++) {
    const line = buf.getLine(y)
    if (!line) continue
    const startX = y === startY ? pos.start.x : 0
    const endX = y === endY ? pos.end.x : undefined
    const text = line.translateToString(true, startX, endX)

    if (y === startY) {
      out = text
      continue
    }

    const prev = buf.getLine(y - 1)
    const join = !!(line.isWrapped || isFullWidthHardWrap(prev, cols))
    out += join ? text : `\n${text}`
  }

  return out.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
}

/** less 等工具按列硬折行时，上一行会写满终端宽度且末列非空 */
function isFullWidthHardWrap(prevLine, cols) {
  if (!prevLine || !cols || cols < 2) return false
  try {
    if (typeof prevLine.getCell === 'function') {
      const last = prevLine.getCell(cols - 1)
      if (last) {
        const chars = last.getChars?.() || ''
        const width = last.getWidth?.()
        // 末列有可见字符，或宽字符占位（width === 0）
        if ((chars && chars !== ' ') || width === 0) return true
      }
    }
    // 退化：整行（含尾部空格）长度达到列宽
    const raw = prevLine.translateToString(false)
    return raw.length >= cols
  } catch {
    return false
  }
}
