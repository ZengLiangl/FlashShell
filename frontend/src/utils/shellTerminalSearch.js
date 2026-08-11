/**
 * 终端缓冲查找：按物理行扫描，不把超长折行拼成巨串。
 * xterm-addon-search 在单行 JSON/日志折满滚动区时会同步拼接整段并卡死主线程。
 *
 * 关键点：translateToString 的「字符下标」≠ 终端单元格列（CJK/宽字符占 2 格）。
 * 选区与 decoration 必须用单元格列，否则 Ctrl+F 高亮会整体错位。
 */

/** 视口内其它匹配高亮（青绿）；当前匹配仍用选区蓝色 */
export const SEARCH_MATCH_STYLE = Object.freeze({
  backgroundColor: '#1e3a3a',
  overviewRulerColor: '#3dccc7',
  borderColor: '#3dccc7',
})

/**
 * 将 BufferLine 转为可搜索字符串，并建立「字符下标 → 单元格列」映射。
 * @param {import('@xterm/xterm').IBufferLine | import('xterm').IBufferLine | null} line
 * @returns {{ text: string, charToCell: number[], endCell: number }}
 */
export function lineToSearchModel(line) {
  if (!line) return { text: '', charToCell: [], endCell: 0 }
  // 优先走 getCell，保证宽字符映射正确
  if (typeof line.getCell === 'function' && typeof line.length === 'number') {
    const parts = []
    const charToCell = []
    let endCell = 0
    for (let col = 0; col < line.length; col++) {
      const cell = line.getCell(col)
      if (!cell) continue
      const width = typeof cell.getWidth === 'function' ? cell.getWidth() : 1
      if (width === 0) continue // 宽字符的右半占位格
      const chars = typeof cell.getChars === 'function' ? (cell.getChars() || '') : ''
      if (!chars) {
        // 与 translateToString(false) 一致：空单元格按空格计入可搜索文本
        charToCell.push(col)
        parts.push(' ')
        endCell = col + Math.max(1, width)
        continue
      }
      for (let i = 0; i < chars.length; i++) {
        charToCell.push(col)
        parts.push(chars[i])
      }
      endCell = col + Math.max(1, width)
    }
    return { text: parts.join(''), charToCell, endCell }
  }

  // 兜底：无 getCell 时只能用字符串下标当列
  const text = line.translateToString?.(false) || ''
  const charToCell = Array.from({ length: text.length }, (_, i) => i)
  return { text, charToCell, endCell: text.length }
}

/**
 * 在已按 (row, col) 升序的匹配列表中定位当前项（≤999，可放心二分）。
 * @param {Array<{ row: number, col: number }>} list
 * @param {{ row: number, col: number } | null | undefined} match
 * @returns {number} 0-based；未找到为 -1
 */
export function indexOfSearchMatch(list, match) {
  if (!match || !list?.length) return -1
  let lo = 0
  let hi = list.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    const m = list[mid]
    if (m.row < match.row || (m.row === match.row && m.col < match.col)) lo = mid + 1
    else if (m.row > match.row || (m.row === match.row && m.col > match.col)) hi = mid - 1
    else return mid
  }
  return -1
}

/**
 * 字符区间 → 单元格起列与宽度
 * @param {{ charToCell: number[], endCell: number }} model
 * @param {number} charStart
 * @param {number} charLen
 */
export function charRangeToCells(model, charStart, charLen) {
  const { charToCell, endCell } = model
  if (!charToCell.length) {
    return { col: Math.max(0, charStart), size: Math.max(1, charLen) }
  }
  const start = Math.max(0, Math.min(charStart, charToCell.length - 1))
  const endChar = Math.max(start, Math.min(charStart + Math.max(0, charLen), charToCell.length))
  const col = charToCell[start] ?? 0
  const endCol = endChar < charToCell.length ? charToCell[endChar] : endCell
  const size = Math.max(1, endCol - col)
  return { col, size }
}

/**
 * @param {import('xterm').Terminal} terminal
 * @param {string} query
 * @param {{ reverse?: boolean, caseSensitive?: boolean }} [options]
 * @returns {{ row: number, col: number, size: number } | null}
 */
export function findInTerminalBuffer(terminal, query, options = {}) {
  const term = String(query || '')
  if (!terminal || !term) return null

  const reverse = !!options.reverse
  const caseSensitive = !!options.caseSensitive
  const needle = caseSensitive ? term : term.toLowerCase()
  const needleLen = needle.length
  if (!needleLen) return null

  const buf = terminal.buffer.active
  const lineCount = buf.length
  if (lineCount <= 0) return null

  const sel = terminal.getSelectionPosition()
  let startRow = 0
  let startCol = 0
  if (sel) {
    if (reverse) {
      startRow = sel.start.y
      startCol = sel.start.x
    } else {
      startRow = sel.end.y
      startCol = sel.end.x
    }
  } else if (reverse) {
    startRow = Math.min(lineCount - 1, buf.viewportY + terminal.rows - 1)
    startCol = terminal.cols
  } else {
    startRow = buf.viewportY
    startCol = 0
  }

  const match = scan(buf, needle, needleLen, caseSensitive, startRow, startCol, reverse, lineCount)
  if (match) return match
  // 绕回另一半缓冲
  if (reverse) {
    if (startRow >= lineCount - 1 && startCol >= terminal.cols) return null
    return scan(buf, needle, needleLen, caseSensitive, lineCount - 1, terminal.cols, true, lineCount, startRow, startCol)
  }
  if (startRow <= 0 && startCol <= 0) return null
  return scan(buf, needle, needleLen, caseSensitive, 0, 0, false, lineCount, startRow, startCol)
}

/**
 * 收集 [startRow, endRow] 闭区间内的匹配（按物理行，有上限）。
 * @returns {Array<{ row: number, col: number, size: number }>}
 */
export function collectMatchesInRows(terminal, query, startRow, endRow, options = {}) {
  const term = String(query || '')
  if (!terminal || !term) return []
  const caseSensitive = !!options.caseSensitive
  const limit = Math.max(1, Number(options.limit) || 80)
  const needle = caseSensitive ? term : term.toLowerCase()
  const needleLen = needle.length
  if (!needleLen) return []

  const buf = terminal.buffer.active
  const from = Math.max(0, startRow)
  const to = Math.min(buf.length - 1, endRow)
  const out = []
  for (let row = from; row <= to && out.length < limit; row++) {
    const model = lineToSearchModel(buf.getLine(row))
    if (!model.text) continue
    const hay = caseSensitive ? model.text : model.text.toLowerCase()
    let pos = 0
    while (out.length < limit) {
      const at = hay.indexOf(needle, pos)
      if (at < 0) break
      const { col, size } = charRangeToCells(model, at, needleLen)
      out.push({ row, col, size })
      pos = at + Math.max(1, needleLen)
    }
  }
  return out
}

/**
 * @param {import('xterm').Terminal} terminal
 * @param {{ row: number, col: number, size: number }} match
 */
export function selectTerminalMatch(terminal, match) {
  if (!terminal || !match) return
  terminal.select(match.col, match.row, match.size)
  const buf = terminal.buffer.active
  const top = buf.viewportY
  const bottom = top + terminal.rows - 1
  if (match.row < top || match.row > bottom) {
    const target = Math.max(0, match.row - Math.floor(terminal.rows / 2))
    if (typeof terminal.scrollToLine === 'function') {
      terminal.scrollToLine(target)
    } else {
      terminal.scrollLines(target - top)
    }
  }
}

/**
 * 仅为当前视口内的匹配打 decoration（数量≈行数，不会卡死）。
 * @param {import('xterm').Terminal} terminal
 * @param {string} query
 * @param {{ row: number, col: number, size: number } | null} [activeMatch] 当前选中项可跳过，避免与选区叠色
 * @param {Array<{ dispose: Function }>} [bucket] 复用的 disposable 列表
 * @returns {Array<{ dispose: Function }>}
 */
export function highlightViewportMatches(terminal, query, activeMatch = null, bucket = []) {
  clearSearchDecorations(bucket)
  if (!terminal || !String(query || '').trim()) return bucket
  if (typeof terminal.registerDecoration !== 'function' || typeof terminal.registerMarker !== 'function') {
    return bucket
  }

  const buf = terminal.buffer.active
  const top = buf.viewportY
  const bottom = top + terminal.rows - 1
  const matches = collectMatchesInRows(terminal, query, top, bottom, { limit: Math.max(80, terminal.rows * 4) })
  const cursorBase = -buf.baseY - buf.cursorY

  for (const m of matches) {
    if (activeMatch && m.row === activeMatch.row && m.col === activeMatch.col) continue
    let marker
    try {
      marker = terminal.registerMarker(cursorBase + m.row)
    } catch {
      continue
    }
    if (!marker) continue
    let decoration
    try {
      decoration = terminal.registerDecoration({
        marker,
        x: m.col,
        width: m.size,
        backgroundColor: SEARCH_MATCH_STYLE.backgroundColor,
        overviewRulerOptions: {
          color: SEARCH_MATCH_STYLE.overviewRulerColor,
          position: 'center',
        },
      })
    } catch {
      marker.dispose?.()
      continue
    }
    if (!decoration) {
      marker.dispose?.()
      continue
    }
    const disposers = [marker]
    disposers.push(decoration.onRender((el) => {
      if (!el.classList.contains('xterm-find-result-decoration')) {
        el.classList.add('xterm-find-result-decoration')
        el.style.outline = `1px solid ${SEARCH_MATCH_STYLE.borderColor}`
      }
    }))
    disposers.push(decoration.onDispose(() => {
      for (const d of disposers) {
        try { d.dispose?.() } catch { /* ignore */ }
      }
    }))
    bucket.push({
      dispose() {
        try { decoration.dispose() } catch { /* ignore */ }
      },
    })
  }
  return bucket
}

/** @param {Array<{ dispose?: Function }>} bucket */
export function clearSearchDecorations(bucket) {
  if (!bucket?.length) return
  for (const d of bucket) {
    try { d.dispose?.() } catch { /* ignore */ }
  }
  bucket.length = 0
}

function scan(buf, needle, needleLen, caseSensitive, startRow, startCol, reverse, lineCount, stopRow, stopCol) {
  if (reverse) {
    for (let row = startRow; row >= 0; row--) {
      if (stopRow != null && row < stopRow) break
      const fromCol = row === startRow ? startCol : Number.POSITIVE_INFINITY
      const limitCol = stopRow != null && row === stopRow ? stopCol : 0
      const hit = findInRow(buf.getLine(row), needle, needleLen, caseSensitive, fromCol, true, limitCol)
      if (hit) return { row, col: hit.col, size: hit.size }
      if (stopRow != null && row === stopRow) break
    }
    return null
  }

  for (let row = startRow; row < lineCount; row++) {
    if (stopRow != null && row > stopRow) break
    const fromCol = row === startRow ? startCol : 0
    const limitCol = stopRow != null && row === stopRow ? stopCol : Number.POSITIVE_INFINITY
    const hit = findInRow(buf.getLine(row), needle, needleLen, caseSensitive, fromCol, false, limitCol)
    if (hit) return { row, col: hit.col, size: hit.size }
    if (stopRow != null && row === stopRow) break
  }
  return null
}

/**
 * fromCol / limitCol 均为「单元格列」，与 terminal.select / selection 一致。
 */
function findInRow(line, needle, needleLen, caseSensitive, fromCol, reverse, limitCol) {
  if (!line) return null
  const model = lineToSearchModel(line)
  if (!model.text) return null
  const hay = caseSensitive ? model.text : model.text.toLowerCase()

  // 将单元格列约束映射到字符下标区间
  const cellToCharStart = (cellCol) => {
    if (!model.charToCell.length) return Math.max(0, cellCol)
    if (cellCol <= 0) return 0
    for (let i = 0; i < model.charToCell.length; i++) {
      if (model.charToCell[i] >= cellCol) return i
    }
    return model.charToCell.length
  }

  if (reverse) {
    const endCell = Math.min(model.endCell, Math.max(0, fromCol))
    const floorCell = Math.max(0, limitCol || 0)
    if (endCell <= floorCell) return null
    const endChar = cellToCharStart(endCell)
    const floorChar = cellToCharStart(floorCell)
    if (endChar <= floorChar) return null
    const at = hay.lastIndexOf(needle, Math.max(floorChar, endChar - needleLen))
    if (at < 0 || at < floorChar) return null
    return charRangeToCells(model, at, needleLen)
  }

  const startCell = Math.max(0, fromCol)
  const ceilCell = Number.isFinite(limitCol) ? Math.min(model.endCell, limitCol) : model.endCell
  if (startCell >= ceilCell) return null
  const startChar = cellToCharStart(startCell)
  const ceilChar = cellToCharStart(ceilCell)
  if (startChar >= ceilChar) return null
  const at = hay.indexOf(needle, startChar)
  if (at < 0 || at >= ceilChar) return null
  return charRangeToCells(model, at, needleLen)
}
