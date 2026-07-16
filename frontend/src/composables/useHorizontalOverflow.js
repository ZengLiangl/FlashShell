import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

const TAB_WIDTH_ESTIMATE = 132
const MORE_BTN_WIDTH = 36

/**
 * 根据容器宽度估算可见项数量，超出部分放入 overflow 列表（保证 activeKey 可见）。
 */
export function useHorizontalOverflow(itemsRef, activeKeyRef, options = {}) {
  const containerRef = ref(null)
  const maxVisible = ref(999)
  const itemWidth = options.itemWidth || TAB_WIDTH_ESTIMATE
  const moreWidth = options.moreWidth || MORE_BTN_WIDTH

  const updateMaxVisible = () => {
    const el = containerRef.value
    if (!el) return
    const w = el.clientWidth
    if (w <= 0) return
    maxVisible.value = Math.max(1, Math.floor((w - moreWidth) / itemWidth))
  }

  let ro = null
  onMounted(() => {
    updateMaxVisible()
    if (containerRef.value && window.ResizeObserver) {
      ro = new ResizeObserver(updateMaxVisible)
      ro.observe(containerRef.value)
    }
    window.addEventListener('resize', updateMaxVisible)
  })
  onUnmounted(() => {
    ro?.disconnect()
    window.removeEventListener('resize', updateMaxVisible)
  })

  watch(itemsRef, () => updateMaxVisible(), { deep: true })

  const split = computed(() => {
    const list = itemsRef.value || []
    const active = activeKeyRef.value
    if (list.length <= maxVisible.value) {
      return { visible: list, overflow: [] }
    }
    const cap = maxVisible.value
    let visible = list.slice(0, cap)
    const activeIdx = list.findIndex((item) => {
      const key = typeof item === 'string' ? item : item?.machineName || item?.id
      return key === active
    })
    if (activeIdx >= cap) {
      visible = [...list.slice(0, cap - 1), list[activeIdx]]
    }
    const visibleKeys = new Set(
      visible.map((item) => (typeof item === 'string' ? item : item?.machineName || item?.id)),
    )
    const overflow = list.filter((item) => {
      const key = typeof item === 'string' ? item : item?.machineName || item?.id
      return !visibleKeys.has(key)
    })
    return { visible, overflow }
  })

  return { containerRef, split, maxVisible }
}
