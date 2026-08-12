/** 简单列表窗口化：按滚动位置切片，减少长列表 DOM */

export const MACHINE_LIST_VIRTUALIZE_AT = 48
export const MACHINE_LIST_ROW_H = 84
export const MACHINE_LIST_OVERSCAN = 6

/**
 * @param {unknown[]} list
 * @param {number} scrollTop
 * @param {number} viewportH
 * @param {{ rowH?: number, overscan?: number, cols?: number }} [opts]
 */
export function windowMachineList(list, scrollTop, viewportH, opts = {}) {
  const items = Array.isArray(list) ? list : []
  const rowH = opts.rowH || MACHINE_LIST_ROW_H
  const overscan = opts.overscan ?? MACHINE_LIST_OVERSCAN
  const cols = Math.max(1, opts.cols || 1)
  if (items.length < MACHINE_LIST_VIRTUALIZE_AT) {
    return { items, start: 0, padTop: 0, padBottom: 0, totalH: 0, virtual: false }
  }
  const rows = Math.ceil(items.length / cols)
  const totalH = rows * rowH
  const viewRows = Math.max(1, Math.ceil((viewportH || 400) / rowH))
  const startRow = Math.max(0, Math.floor((scrollTop || 0) / rowH) - overscan)
  const endRow = Math.min(rows, startRow + viewRows + overscan * 2)
  const start = startRow * cols
  const end = Math.min(items.length, endRow * cols)
  return {
    items: items.slice(start, end),
    start,
    padTop: startRow * rowH,
    padBottom: Math.max(0, totalH - endRow * rowH),
    totalH,
    virtual: true,
  }
}
