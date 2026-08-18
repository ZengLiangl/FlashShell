/**
 * Shell 切 tab 时监控 / SFTP 画面保留策略。
 * 组件只负责存取；是否清零、是否拆树由这里决定，便于回归。
 */

export function hasUsefulMonitorSnapshot(snap) {
  if (!snap) return false
  const memTotal = String(snap.memTotal || '').trim()
  if (memTotal && memTotal !== '0') return true
  if ((Number(snap.cpuPercent) || 0) > 0) return true
  if ((Number(snap.memPercent) || 0) > 0) return true
  const up = String(snap.uptimeText || '').trim()
  if (up && up !== '0') return true
  return (snap.topMem || []).length > 0
}

/**
 * 过期轮询必须丢弃，且不得把当前画面清零。
 */
export function shouldDiscardMonitorResult({ idle, activeMachine, machineAtStart }) {
  return !!(idle || activeMachine !== machineAtStart)
}

/**
 * 空快照 / 辅助通道缺失时，只保住「当前这台机器」的上次有效画面。
 * 上一台机器残留的数字不能继续显示在新 tab 上。
 */
export function shouldKeepCurrentMonitorSnapshot({
  current,
  activeMachine,
  incoming,
  auxMissing = false,
}) {
  const incomingEmpty = auxMissing || !hasUsefulMonitorSnapshot(incoming)
  if (!incomingEmpty) return false
  return hasUsefulMonitorSnapshot(current)
    && String(current.machineName || '') === String(activeMachine || '')
}

/**
 * 断开或空画面不要覆盖该会话已有的有效缓存。
 */
export function shouldReplaceMonitorCache({ current, hasExisting }) {
  if (hasUsefulMonitorSnapshot(current)) return true
  return !hasExisting
}

export function canRestoreSftpBrowseView(cached) {
  return !!(cached && cached.cwd)
}

/**
 * 切回已逛过的会话且目录未变时，只刷新文件列表，不从 / 重建左侧树。
 */
export function shouldSilentSftpRefresh({ restored, cwd, target, treeHasNodes }) {
  return !!(restored && treeHasNodes && cwd && target && cwd === target)
}
