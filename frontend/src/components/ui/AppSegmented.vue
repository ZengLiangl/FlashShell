<template>
  <div class="seg" role="tablist" :aria-label="ariaLabel">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      role="tab"
      :class="{ active: modelValue === opt.value }"
      :aria-selected="modelValue === opt.value"
      @click="$emit('update:modelValue', opt.value)"
    >
      <component :is="opt.icon" v-if="opt.icon" class="seg-icon" />
      <span>{{ opt.label }}</span>
      <span
        v-if="opt.dot && (opt.dotActive ?? modelValue === opt.value)"
        class="seg-dot dot on"
        aria-hidden="true"
      />
    </button>
  </div>
</template>

<script>
export default {
  name: 'AppSegmented',
  props: {
    modelValue: { type: String, default: '' },
    options: { type: Array, default: () => [] },
    ariaLabel: { type: String, default: '模式切换' },
  },
  emits: ['update:modelValue'],
}
</script>

<style scoped>
.seg-icon {
  width: 13px;
  height: 13px;
}
.seg-dot {
  width: 6px;
  height: 6px;
}
</style>
