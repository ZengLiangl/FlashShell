<template>
  <button
    type="button"
    class="icon-btn"
    :class="{ badge, active }"
    :data-count="badgeCount > 0 ? badgeCount : undefined"
    :title="title"
    :aria-label="ariaLabel || title"
    :disabled="disabled"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<script>
export default {
  name: 'AppIconBtn',
  props: {
    title: { type: String, default: '' },
    ariaLabel: { type: String, default: '' },
    badge: { type: Boolean, default: false },
    badgeCount: { type: [Number, String], default: 0 },
    active: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  emits: ['click'],
}
</script>
