/** xterm 滚动缓冲行数上限（默认，可被系统设置覆盖） */
export const SHELL_TERMINAL_SCROLLBACK = 2000

/** 任务执行终端输出行数上限（默认） */
export const TASK_OUTPUT_MAX_LINES = 1000

/** Shell 命令历史每作用域条数上限（默认） */
export const SHELL_COMMAND_HISTORY_MAX = 200

/** 活跃 Shell 标签输出缓冲上限（字节） */
export const SHELL_OUTPUT_BUFFER_ACTIVE_MAX = 2 * 1024 * 1024

/** 后台 Shell 标签输出缓冲上限（字节） */
export const SHELL_OUTPUT_BUFFER_BACKGROUND_MAX = 512 * 1024

export function clampShellTerminalScrollback(n) {
  const v = Number(n)
  if (!Number.isFinite(v) || v <= 0) return SHELL_TERMINAL_SCROLLBACK
  return Math.min(100000, Math.max(100, Math.round(v)))
}

export function clampTaskOutputMaxLines(n) {
  const v = Number(n)
  if (!Number.isFinite(v) || v <= 0) return TASK_OUTPUT_MAX_LINES
  return Math.min(100000, Math.max(100, Math.round(v)))
}

export function clampShellCommandHistoryMax(n) {
  const v = Number(n)
  if (!Number.isFinite(v) || v <= 0) return SHELL_COMMAND_HISTORY_MAX
  return Math.min(20000, Math.max(50, Math.round(v)))
}
