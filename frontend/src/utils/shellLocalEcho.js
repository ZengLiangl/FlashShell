/**
 * SSH 本地回显（输入体验优化，不是 X11 转发）。
 * 可打印字符由客户端立即写入终端，远端相同回显被丢掉，避免双字符。
 * 退格在仍有未确认预测时本地擦除；方向键 / Tab / 回车等控制序列交给远端。
 */

const BACKSPACE_ECHO_PATTERNS = [
  '\b \b',
  '\b\x1b[K',
  '\x08\x1b[K',
  '\x1b[D\x1b[K',
  '\x1b[D\x1b[P',
  '\b',
]

export function createLocalEchoState() {
  return { pending: '', backspaces: 0 }
}

export function resetLocalEchoState(state) {
  if (!state) return
  state.pending = ''
  state.backspaces = 0
}

function popLastCodePoint(s) {
  const chars = Array.from(String(s || ''))
  if (!chars.length) return ''
  chars.pop()
  return chars.join('')
}

function nextCodePoint(text, i) {
  const s = String(text || '')
  if (i >= s.length) return ''
  const code = s.charCodeAt(i)
  if (code >= 0xd800 && code <= 0xdbff && i + 1 < s.length) {
    const next = s.charCodeAt(i + 1)
    if (next >= 0xdc00 && next <= 0xdfff) return s.slice(i, i + 2)
  }
  return s[i]
}

/** 取出从 i 起的完整 ESC 序列；不完整则吃到末尾。 */
export function takeAnsiSeq(text, i) {
  const s = String(text || '')
  if (i >= s.length || s.charCodeAt(i) !== 0x1b) return ''
  const n = s.length
  const n1 = s[i + 1]
  if (n1 === ']') {
    for (let j = i + 2; j < n; j++) {
      if (s[j] === '\x07') return s.slice(i, j + 1)
      if (s[j] === '\x1b' && s[j + 1] === '\\') return s.slice(i, j + 2)
    }
    return s.slice(i)
  }
  if (n1 === '[') {
    let j = i + 2
    while (j < n) {
      const c = s.charCodeAt(j)
      if (c >= 0x40 && c <= 0x7e) return s.slice(i, j + 1)
      j += 1
    }
    return s.slice(i)
  }
  if (n1 === 'O' && i + 2 < n) return s.slice(i, i + 3)
  if (i + 1 < n) return s.slice(i, i + 2)
  return s.slice(i)
}

function isPassthroughSeq(seq) {
  if (!seq || seq.charCodeAt(0) !== 0x1b) return false
  if (seq[1] === ']') return true
  return seq[1] === '[' && seq.endsWith('m')
}

function eatBackspaceEcho(text, i) {
  const s = String(text || '')
  for (const pat of BACKSPACE_ECHO_PATTERNS) {
    if (s.startsWith(pat, i)) return pat.length
  }
  return 0
}

/**
 * 根据键盘输入生成本地显示，并更新预测队列。
 * @returns {{ display: string }}
 */
export function applyLocalEchoInput(data, state) {
  if (!state) return { display: '' }
  const raw = String(data || '')
  let display = ''
  let i = 0
  while (i < raw.length) {
    const ch = raw[i]
    const code = raw.charCodeAt(i)
    if (ch === '\r' || ch === '\n') {
      break
    }
    if (ch === '\x1b' || ch === '\t') {
      resetLocalEchoState(state)
      break
    }
    if (ch === '\x03' || ch === '\x04' || ch === '\x0c' || ch === '\x15' || ch === '\x17' || ch === '\x1a') {
      resetLocalEchoState(state)
      break
    }
    if (ch === '\x7f' || ch === '\b') {
      if (state.pending) {
        state.pending = popLastCodePoint(state.pending)
        display += '\b \b'
        state.backspaces += 1
      } else {
        resetLocalEchoState(state)
      }
      i += 1
      continue
    }
    if (code < 32) {
      resetLocalEchoState(state)
      break
    }
    const one = nextCodePoint(raw, i)
    display += one
    state.pending += one
    i += one.length
  }
  return { display }
}

/**
 * 丢掉已本地显示的远端回显，避免重复字符。
 * @returns {string} 仍需写入终端的文本
 */
export function applyRemoteEchoSuppression(incoming, state) {
  const raw = String(incoming || '')
  if (!state || (!state.pending && !state.backspaces) || !raw) return raw
  let out = ''
  let i = 0
  while (i < raw.length) {
    if (state.backspaces > 0) {
      const seq = takeAnsiSeq(raw, i)
      if (seq && isPassthroughSeq(seq)) {
        out += seq
        i += seq.length
        continue
      }
      const n = eatBackspaceEcho(raw, i)
      if (n > 0) {
        state.backspaces -= 1
        i += n
        continue
      }
    }
    const seq = takeAnsiSeq(raw, i)
    if (seq) {
      if (isPassthroughSeq(seq)) {
        out += seq
        i += seq.length
        continue
      }
      resetLocalEchoState(state)
      out += raw.slice(i)
      return out
    }
    if (!state.pending) {
      out += raw.slice(i)
      break
    }
    const one = nextCodePoint(raw, i)
    const expect = nextCodePoint(state.pending, 0)
    if (one && one === expect) {
      state.pending = state.pending.slice(one.length)
      i += one.length
      continue
    }
    resetLocalEchoState(state)
    out += raw.slice(i)
    break
  }
  return out
}
