<template>
  <div class="progress" role="progressbar" :aria-valuenow="clamped" aria-valuemin="0" aria-valuemax="100">
    <i :style="{ width: `${clamped}%` }" />
  </div>
</template>

<script>
import { computed } from 'vue'

export default {
  name: 'AppProgress',
  props: {
    value: { type: Number, default: 0 },
  },
  setup(props) {
    const clamped = computed(() => Math.max(0, Math.min(100, Number(props.value) || 0)))
    return { clamped }
  },
}
</script>

<style scoped>
.progress {
  flex: 1;
  max-width: 320px;
  height: 5px;
  border-radius: 3px;
  background: var(--surface-2);
  overflow: hidden;
}
.progress i {
  display: block;
  height: 100%;
  background: var(--accent);
  border-radius: 3px;
  transition: width 0.3s ease;
}
</style>
