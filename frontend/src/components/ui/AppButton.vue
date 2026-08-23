<template>
  <button
    type="button"
    class="btn"
    :class="[variant, size, { 'is-loading': loading }]"
    :disabled="disabled || loading"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<script>
export default {
  name: 'AppButton',
  props: {
    variant: {
      type: String,
      default: 'default',
      validator: (v) => ['default', 'primary', 'danger', 'soft', 'ghost'].includes(v),
    },
    size: {
      type: String,
      default: 'md',
      validator: (v) => ['md', 'sm'].includes(v),
    },
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
}
</script>

<style scoped>
.btn.is-loading {
  opacity: 0.65;
  pointer-events: none;
}
.btn-sm {
  height: 24px;
  padding: 0 9px;
  font-size: 12.5px;
  border-radius: 6px;
}
.btn-primary {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--on-accent);
}
.btn-primary:hover {
  background: var(--accent-strong);
  border-color: var(--accent-strong);
  color: var(--on-accent);
}
.btn-danger {
  color: var(--danger);
  border-color: color-mix(in oklch, var(--danger) 30%, var(--border));
}
.btn-danger:hover {
  background: var(--danger-soft);
  border-color: var(--danger);
  color: var(--danger);
}
.btn-soft {
  background: var(--accent-soft);
  border-color: transparent;
  color: var(--accent);
}
.btn-soft:hover {
  background: color-mix(in oklch, var(--accent) 22%, transparent);
  color: var(--accent-strong);
}
.btn-ghost {
  border-color: transparent;
  background: transparent;
}
.btn-ghost:hover {
  background: var(--surface-2);
}
</style>
