/**
 * 判断是否存在比「系统设置」更上层的弹层（子 Dialog / MessageBox 等）。
 * 用于 Escape：只关最上层，不连带关掉设置壳。
 */
export function hasOverlayAboveSettingsHub() {
  if (typeof document === 'undefined') return false

  // MessageBox / Message / Notification 等独立弹层
  if (
    document.querySelector(
      '.el-message-box, .el-overlay.is-message-box, .el-popconfirm, .el-image-viewer__wrapper',
    )
  ) {
    return true
  }

  const overlays = Array.from(document.querySelectorAll('body > .el-overlay')).filter((el) => {
    const style = window.getComputedStyle(el)
    return (
      style.display !== 'none' &&
      style.visibility !== 'hidden' &&
      Number.parseFloat(style.opacity || '1') > 0.01
    )
  })

  // 系统设置本身占 1 层；>1 说明还有子 Dialog（append-to-body）
  return overlays.length > 1
}
