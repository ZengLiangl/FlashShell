/**
 * 终端缓冲查找：按物理行扫描，不把超长折行拼成巨串。
 * xterm-addon-search 在单行 JSON/日志折满滚动区时会同步拼接整段并卡死主线程。
 */

/** 视口内其它匹配高亮（青绿）；当前匹配仍用选区蓝色 */
export const SEARCH_MATCH_STYLE = Object.freeze({
  backgroundColor: '#1e3a3a',
  overviewRulerColor: '#3dccc7',
  borderColor: '#3dccc7',
})

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
    const line = buf.getLine(row)
    if (!line) continue
    const raw = line.translateToString(false)
    if (!raw) continue
    const hay = caseSensitive ? raw : raw.toLowerCase()
    let pos = 0
    while (out.length < limit) {
      const at = hay.indexOf(needle, pos)
      if (at < 0) break
      out.push({ row, col: at, size: Math.min(needleLen, hay.length - at) })
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

function findInRow(line, needle, needleLen, caseSensitive, fromCol, reverse, limitCol) {
  if (!line) return null
  // 只用当前物理行，绝不向上/下吞折行，避免单行数百 KB JSON 拼串卡死
  const raw = line.translateToString(false)
  if (!raw) return null
  const hay = caseSensitive ? raw : raw.toLowerCase()

  if (reverse) {
    const end = Math.min(hay.length, Math.max(0, fromCol))
    const floor = Math.max(0, limitCol || 0)
    if (end <= floor) return null
    const at = hay.lastIndexOf(needle, Math.max(floor, end - needleLen))
    if (at < 0 || at < floor) return null
    return { col: at, size: Math.min(needleLen, hay.length - at) }
  }

  const start = Math.max(0, fromCol)
  const ceil = Number.isFinite(limitCol) ? Math.min(hay.length, limitCol) : hay.length
  if (start >= ceil) return null
  const at = hay.indexOf(needle, start)
  if (at < 0 || at >= ceil) return null
  return { col: at, size: Math.min(needleLen, hay.length - at) }
}
