<template>
  <el-tooltip
    :content="displayText"
    placement="top"
    :show-after="300"
    :disabled="!overflowing"
  >
    <span
      ref="elRef"
      class="text-overflow-tooltip"
      :class="textClass"
      @mouseenter="checkOverflow"
    >
      <slot>{{ displayText }}</slot>
    </span>
  </el-tooltip>
</template>

<script>
import { computed, ref } from 'vue'

export default {
  name: 'TextOverflowTooltip',
  props: {
    text: { type: String, default: '' },
    textClass: { type: [String, Array, Object], default: '' },
  },
  setup(props) {
    const elRef = ref(null)
    const overflowing = ref(false)

    const displayText = computed(() => String(props.text ?? ''))

    const checkOverflow = () => {
      const el = elRef.value
      if (!el || !displayText.value) {
        overflowing.value = false
        return
      }
      overflowing.value = el.scrollWidth > el.clientWidth + 1
    }

    return {
      elRef,
      overflowing,
      displayText,
      checkOverflow,
    }
  },
}
</script>

<style scoped>
.text-overflow-tooltip {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
