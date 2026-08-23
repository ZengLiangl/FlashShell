<template>
  <div class="search" :class="{ compact }">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
      <circle cx="11" cy="11" r="7" />
      <path d="M21 21l-4.3-4.3" />
    </svg>
    <input
      ref="inputRef"
      :value="modelValue"
      type="search"
      :placeholder="placeholder"
      :aria-label="ariaLabel || placeholder"
      @input="onInput"
      @keydown.enter.exact.prevent="$emit('enter', $event)"
      @keydown.esc.prevent="$emit('escape', $event)"
    />
    <span v-if="shortcutHint" class="kbd">{{ shortcutHint }}</span>
  </div>
</template>

<script>
import { ref } from 'vue'

export default {
  name: 'AppSearch',
  props: {
    modelValue: { type: String, default: '' },
    placeholder: { type: String, default: '搜索…' },
    ariaLabel: { type: String, default: '' },
    shortcutHint: { type: String, default: '' },
    compact: { type: Boolean, default: false },
  },
  emits: ['update:modelValue', 'enter', 'escape'],
  setup(props, { emit, expose }) {
    const inputRef = ref(null)
    const onInput = (e) => emit('update:modelValue', e.target.value)
    const focus = () => inputRef.value?.focus?.()
    expose({ focus })
    return { inputRef, onInput, focus }
  },
}
</script>

<style scoped>
.search.compact {
  max-width: 280px;
}
</style>
