/**
 * 任务模式执行进度：顶部/底部/进度条共用同一套步数口径。
 * 运行中按「当前正在执行的步」（1 起算）展示，与终端日志「执行步骤 N」一致。
 */

/**
 * @param {object|null|undefined} status
 * @returns {{ completed: number, total: number }}
 */
export function getTaskStepCounts(status) {
  const total = Math.max(0, Number(status?.totalSteps) || 0)
  const done = Math.max(0, Number(status?.completedSteps) || 0)
  const clampedDone = total > 0 ? Math.min(done, total) : done
  let current = clampedDone
  if (status?.isRunning && total > 0) {
    current = Math.min(clampedDone + 1, total)
  }
  return {
    completed: current,
    total,
  }
}

/**
 * @param {object|null|undefined} status
 * @returns {number} 0–100
 */
export function calcTaskProgressPercentage(status) {
  if (!status?.isRunning) return 0
  const { completed, total } = getTaskStepCounts(status)
  if (total <= 0) return 0
  return Math.min(100, Math.round((completed / total) * 100))
}

/**
 * @param {object|null|undefined} status
 * @returns {string} 如 `2/3 步`；无总步数时为空串
 */
export function formatTaskStepProgress(status) {
  const { completed, total } = getTaskStepCounts(status)
  if (total <= 0) return ''
  return `${completed}/${total} 步`
}
