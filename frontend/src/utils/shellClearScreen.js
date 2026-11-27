/**
 * 把「清屏」补成「同时清回滚缓冲」。
 *
 * 背景：shell 的 clear 命令到底发什么由远端 terminfo 决定。
 * 实测 hk-test-118 / hk-test-136 的 xterm-256color 都没有 E3 能力，
 * clear 只发 ESC[H ESC[2J —— xterm.js 的 2J 仅擦除视口，回滚缓冲原样保留，
 * 于是 clear 之后向上滚还能看到旧内容（用户感知为「清屏没生效」）。
 *
 * 这里把裸的 2J 改写成 2J + 3J（xterm.js 支持 3J 清回滚）。
 * 备用缓冲区（vim/less/top 的 alt screen）没有回滚，3J 是空操作，因此不影响 TUI。
 *
 * 注意：ESC[?2J 这类带私有前缀的序列不处理；跨数据块被截断的序列会在下一次拼接。
 */

const ESC = '\x1b'
// 尾部可能是未完成的「清屏序列前缀」，先攒着等下一块
const TAIL_PATTERN = /\x1b(?:\[[0-9;?]{0,3})?$/
const MAX_TAIL = 6

/**
 * @param {string} text 本次解码出的文本
 * @param {{ tail: string }} state 跨块状态（调用方持有，初值 { tail: '' }）
 * @returns {string} 需要写入终端的文本
 */
export function rewriteClearScreen(text, state) {
  if (!state || typeof state !== 'object') return text ?? ''
  if (!text) return ''
  const merged = (state.tail || '') + text
  state.tail = ''
  if (!merged) return ''

  let body = merged
  const tail = merged.match(TAIL_PATTERN)
  if (tail && tail[0] && tail[0].length <= MAX_TAIL && tail[0] !== merged) {
    state.tail = tail[0]
    body = merged.slice(0, merged.length - state.tail.length)
  }
  if (!body) return ''

  // 只处理裸 2J：ESC[2J → ESC[2J ESC[3J
  return body.split(`${ESC}[2J`).join(`${ESC}[2J${ESC}[3J`)
}

/** 新建一份跨块状态 */
export function createClearScreenState() {
  return { tail: '' }
}
