<template>
  <span class="dot" :class="state" aria-hidden="true" />
</template>

<script>
export default {
  name: 'StatusDot',
  props: {
    state: {
      type: String,
      default: 'off',
      validator: (v) => ['on', 'off', 'warn', 'err'].includes(v),
    },
  },
}
</script>
