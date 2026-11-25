/**
 * 终端缓冲查找。
 *
 * 不用 xterm-addon-search：它会把超长折行（grep 出的整行 JSON）拼成巨串同步扫描，容易卡死。
 * 这里按「逻辑行」扫描：把 isWrapped 的物理行拼起来再找，既能搜到跨行折断的关键字
 * （grep/less 的长行），又只处理单条逻辑行，配合时间分片不会阻塞主线程。
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

/** 单条逻辑行最多拼接的字符数，防止极端一行 JSON 撑爆内存 */
const MAX_LOGICAL_CHARS = 500000

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
 * 把从 row 开始的逻辑行（连续 isWrapped 的物理行）拼成一个可搜索字符串。
 * segments 保留每一物理行的字符区间与单元格映射，用于把匹配还原成 (row, col, size)。
 *
 * @param {import('xterm').IBuffer} buf
 * @param {number} row
 * @param {number} [maxChars]
 * @returns {{ text: string, segments: Array<{ row: number, startChar: number, charToCell: number[], endCell: number }>, startRow: number, endRow: number, truncated: boolean }}
 */
export function buildLogicalLine(buf, row, maxChars = MAX_LOGICAL_CHARS) {
  let startRow = Math.max(0, Math.min(row, buf.length - 1))
  // 若当前行是折行续行，回溯到逻辑行首行
  while (startRow > 0) {
    const line = buf.getLine(startRow)
    if (!line || !line.isWrapped) break
    startRow -= 1
  }

  const segments = []
  let text = ''
  let r = startRow
  let truncated = false
  while (r < buf.length) {
    const line = buf.getLine(r)
    if (!line) break
    if (r > startRow && !line.isWrapped) break
    if (truncated) {
      // 超长逻辑行：不再拼接内容，但继续走到真正的行尾，
      // 否则调用方按 endRow+1 前进时会因为 isWrapped 又回溯到同一行，死循环
      r += 1
      continue
    }
    const model = lineToSearchModel(line)
    if (text.length + model.text.length > maxChars) {
      truncated = true
      r += 1
      continue
    }
    segments.push({
      row: r,
      startChar: text.length,
      charToCell: model.charToCell,
      endCell: model.endCell,
    })
    text += model.text
    r += 1
  }
  return { text, segments, startRow, endRow: Math.max(startRow, r - 1), truncated }
}

/** 字符下标 → { row, col } */
function charIndexToCell(logical, charIndex) {
  const { segments, text } = logical
  if (!segments.length) return { row: logical.startRow, col: 0 }
  if (charIndex <= 0) {
    const first = segments[0]
    return { row: first.row, col: first.charToCell[0] ?? 0 }
  }
  if (charIndex >= text.length) {
    const last = segments[segments.length - 1]
    return { row: last.row, col: last.endCell }
  }
  let lo = 0
  let hi = segments.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    const seg = segments[mid]
    if (charIndex < seg.startChar) {
      hi = mid - 1
    } else if (charIndex >= seg.startChar + seg.charToCell.length) {
      lo = mid + 1
    } else {
      const local = charIndex - seg.startChar
      return { row: seg.row, col: seg.charToCell[local] ?? seg.endCell }
    }
  }
  const last = segments[segments.length - 1]
  return { row: last.row, col: last.endCell }
}

/** (row, col) → 字符下标（col 取第一个 >= 该列的字符） */
function cellToCharIndex(logical, row, col) {
  const seg = logical.segments.find((item) => item.row === row)
  if (!seg) return logical.text.length
  const cells = seg.charToCell
  if (!cells.length) return seg.startChar
  let lo = 0
  let hi = cells.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (cells[mid] < col) lo = mid + 1
    else hi = mid - 1
  }
  return seg.startChar + lo
}

/** 字符区间 → 匹配（含跨行 segments，供高亮逐行绘制） */
function matchFromCharRange(logical, startChar, needleLen, cols) {
  const startPos = charIndexToCell(logical, startChar)
  const endPos = charIndexToCell(logical, startChar + needleLen)
  const columns = Math.max(1, cols)
  const size = (endPos.row - startPos.row) * columns + (endPos.col - startPos.col)
  const segments = []
  for (let row = startPos.row; row <= endPos.row; row++) {
    const segStart = row === startPos.row ? startPos.col : 0
    const segEnd = row === endPos.row ? endPos.col : columns
    if (segEnd > segStart) segments.push({ row, col: segStart, size: segEnd - segStart })
  }
  return { row: startPos.row, col: startPos.col, size: Math.max(1, size), segments }
}

/**
 * 在单条逻辑行内查找。
 * @param {object} logical
 * @param {string} needle 已按大小写处理过的关键词
 * @param {number} needleLen
 * @param {boolean} caseSensitive
 * @param {{ fromChar?: number, endLimit?: number }} options
 *   fromChar：正向起点（含）；endLimit：正向截止（不含）/ 反向匹配结束位置上限
 * @param {number} cols 终端列数
 */
function findInLogicalLine(logical, needle, needleLen, caseSensitive, options = {}, cols) {
  const text = logical.text
  if (!text || !needleLen) return null
  const hay = caseSensitive ? text : text.toLowerCase()
  const endLimit = options.endLimit == null ? hay.length : Math.max(0, Math.min(hay.length, options.endLimit))

  if (options.fromChar != null) {
    const from = Math.max(0, Math.min(hay.length, options.fromChar))
    if (from >= endLimit) return null
    const at = hay.indexOf(needle, from)
    if (at < 0 || at + needleLen > endLimit) return null
    return matchFromCharRange(logical, at, needleLen, cols)
  }

  const lastStart = Math.min(endLimit - needleLen, hay.length - needleLen)
  if (lastStart < 0) return null
  const at = hay.lastIndexOf(needle, lastStart)
  if (at < 0 || at + needleLen > endLimit) return null
  return matchFromCharRange(logical, at, needleLen, cols)
}

/**
 * 在一条逻辑行里收集所有匹配（有上限，供异步计数/视口高亮使用）。
 * @returns {Array<{ row: number, col: number, size: number, segments: Array<{row:number,col:number,size:number}> }>}
 */
export function collectLogicalLineMatches(logical, query, options = {}) {
  const text = String(query || '')
  if (!logical?.text || !text) return []
  const cols = Math.max(1, Number(options.cols) || 1)
  const limit = Math.max(1, Number(options.limit) || 80)
  const caseSensitive = !!options.caseSensitive
  const needle = caseSensitive ? text : text.toLowerCase()
  const hay = caseSensitive ? logical.text : logical.text.toLowerCase()
  const out = []
  let from = 0
  while (out.length < limit) {
    const at = hay.indexOf(needle, from)
    if (at < 0) break
    out.push(matchFromCharRange(logical, at, needle.length, cols))
    from = at + Math.max(1, needle.length)
  }
  return out
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
 * 字符区间 → 单元格起列与宽度（保留给单行场景）
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

/** 匹配的结束位置（下一个待搜起点），供「下一处」避免重复/跳跃 */
export function matchEndPosition(match, cols) {
  if (!match) return null
  const columns = Math.max(1, cols)
  const total = match.col + match.size
  return { row: match.row + Math.floor(total / columns), col: total % columns }
}

/**
 * 在终端缓冲里查找下一处/上一处。
 * options.start 用 { row, col } 明确起点（占位符语义见 cellToCharIndex）：
 *  - 正向：从该单元格起（含）向后找
 *  - 反向：从该单元格往前找（匹配结束位置 <= 该单元格）
 * 不传 start 时正向从视口顶部、反向从视口底部开始。
 *
 * @param {import('xterm').Terminal} terminal
 * @param {string} query
 * @param {{ reverse?: boolean, caseSensitive?: boolean, start?: { row: number, col: number } }} [options]
 */
export function findInTerminalBuffer(terminal, query, options = {}) {
  const text = String(query || '')
  if (!terminal || !text) return null

  const reverse = !!options.reverse
  const caseSensitive = !!options.caseSensitive
  const needle = caseSensitive ? text : text.toLowerCase()
  const needleLen = needle.length
  if (!needleLen) return null

  const buf = terminal.buffer.active
  const total = buf.length
  if (total <= 0) return null
  const cols = Math.max(1, terminal.cols)

  const rawStart = options.start || (reverse
    ? { row: Math.min(total - 1, buf.viewportY + terminal.rows - 1), col: cols }
    : { row: buf.viewportY, col: 0 })
  const startRow = Math.max(0, Math.min(total - 1, Number(rawStart.row) || 0))
  const startCol = Math.max(0, Math.min(cols, Number(rawStart.col) || 0))

  const startLine = buildLogicalLine(buf, startRow)

  if (reverse) {
    const startChar = cellToCharIndex(startLine, startRow, startCol)
    let hit = findInLogicalLine(startLine, needle, needleLen, caseSensitive, { endLimit: startChar }, cols)
    if (hit) return hit

    // 往前扫
    let row = startLine.startRow - 1
    while (row >= 0) {
      const logical = buildLogicalLine(buf, row)
      hit = findInLogicalLine(logical, needle, needleLen, caseSensitive, {}, cols)
      if (hit) return hit
      row = logical.startRow - 1
    }

    // 绕回缓冲区末尾，扫到起始逻辑行之后
    row = total - 1
    while (row > startLine.endRow) {
      const logical = buildLogicalLine(buf, row)
      hit = findInLogicalLine(logical, needle, needleLen, caseSensitive, {}, cols)
      if (hit) return hit
      row = logical.startRow - 1
    }
    return null
  }

  const startChar = cellToCharIndex(startLine, startRow, startCol)
  let hit = findInLogicalLine(startLine, needle, needleLen, caseSensitive, { fromChar: startChar }, cols)
  if (hit) return hit

  // 往后扫
  let row = startLine.endRow + 1
  while (row < total) {
    const logical = buildLogicalLine(buf, row)
    hit = findInLogicalLine(logical, needle, needleLen, caseSensitive, {}, cols)
    if (hit) return hit
    row = logical.endRow + 1
  }

  // 绕回开头，扫到起始逻辑行之前
  row = 0
  while (row < startLine.startRow) {
    const logical = buildLogicalLine(buf, row)
    hit = findInLogicalLine(logical, needle, needleLen, caseSensitive, {}, cols)
    if (hit) return hit
    row = logical.endRow + 1
  }

  // 同一条逻辑行里、起点之前的部分
  hit = findInLogicalLine(startLine, needle, needleLen, caseSensitive, { fromChar: 0, endLimit: startChar }, cols)
  return hit || null
}

/**
 * 收集与 [startRow, endRow] 有交集的匹配（按逻辑行，上限保护）。
 * @returns {Array<{ row: number, col: number, size: number, segments: Array<{row:number,col:number,size:number}> }>}
 */
export function collectMatchesInRows(terminal, query, startRow, endRow, options = {}) {
  const text = String(query || '')
  if (!terminal || !text) return []
  const buf = terminal.buffer.active
  if (buf.length <= 0) return []
  const cols = Math.max(1, terminal.cols)
  const limit = Math.max(1, Number(options.limit) || 80)
  const caseSensitive = !!options.caseSensitive

  const from = Math.max(0, Math.min(startRow, buf.length - 1))
  const to = Math.max(from, Math.min(endRow, buf.length - 1))
  let row = from
  while (row > 0 && buf.getLine(row)?.isWrapped) row -= 1

  const out = []
  while (row <= to && row < buf.length && out.length < limit) {
    const logical = buildLogicalLine(buf, row)
    const matches = collectLogicalLineMatches(logical, text, {
      caseSensitive,
      limit: limit - out.length,
      cols,
    })
    for (const match of matches) {
      const touchesRange = match.segments?.some((seg) => seg.row >= from && seg.row <= to)
      if (touchesRange) out.push(match)
      if (out.length >= limit) break
    }
    row = logical.endRow + 1
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
 * 跨行匹配按行拆成多个 decoration，避免长关键字只亮半截。
 *
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
  const cols = Math.max(1, terminal.cols)
  const top = buf.viewportY
  const bottom = top + terminal.rows - 1
  const matches = collectMatchesInRows(terminal, query, top, bottom, {
    limit: Math.max(80, terminal.rows * 4),
  })
  const cursorBase = -buf.baseY - buf.cursorY

  for (const match of matches) {
    if (
      activeMatch &&
      match.row === activeMatch.row &&
      match.col === activeMatch.col &&
      match.size === activeMatch.size
    ) {
      continue
    }
    const segments = match.segments?.length
      ? match.segments
      : [{ row: match.row, col: match.col, size: match.size }]
    for (const seg of segments) {
      if (seg.row < top || seg.row > bottom) continue
      const width = Math.min(seg.size, cols - seg.col)
      if (width <= 0) continue

      let marker
      try {
        marker = terminal.registerMarker(cursorBase + seg.row)
      } catch {
        continue
      }
      if (!marker) continue
      let decoration
      try {
        decoration = terminal.registerDecoration({
          marker,
          x: seg.col,
          width,
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
          try {
            d.dispose?.()
          } catch {
            /* ignore */
          }
        }
      }))
      bucket.push({
        dispose() {
          try {
            decoration.dispose()
          } catch {
            /* ignore */
          }
        },
      })
    }
  }
  return bucket
}

/** @param {Array<{ dispose?: Function }>} bucket */
export function clearSearchDecorations(bucket) {
  if (!bucket?.length) return
  for (const d of bucket) {
    try {
      d.dispose?.()
    } catch {
      /* ignore */
    }
  }
  bucket.length = 0
}
