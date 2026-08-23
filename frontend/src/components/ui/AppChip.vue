<template>
  <button
    type="button"
    class="chip"
    :class="{ active }"
    @click="$emit('click', $event)"
  >
    <StatusDot v-if="showDot" :state="dotState" />
    <slot />
  </button>
</template>

<script>
import StatusDot from './StatusDot.vue'

export default {
  name: 'AppChip',
  components: { StatusDot },
  props: {
    active: { type: Boolean, default: false },
    showDot: { type: Boolean, default: false },
    dotState: { type: String, default: 'on' },
  },
  emits: ['click'],
}
</script>
