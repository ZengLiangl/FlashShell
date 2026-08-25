/**
 * 任务模式执行进度：顶部/底部/进度条共用同一套「已完成步数」口径。
 * 不把「当前正在跑的步」计入完成数，避免末步一开始就显示接近 100%。
 */

/**
 * @param {object|null|undefined} status
 * @returns {{ completed: number, total: number }}
 */
export function getTaskStepCounts(status) {
  const total = Math.max(0, Number(status?.totalSteps) || 0)
  const completed = Math.max(0, Number(status?.completedSteps) || 0)
  return {
    completed: total > 0 ? Math.min(completed, total) : completed,
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
