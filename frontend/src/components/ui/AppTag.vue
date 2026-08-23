<template>
  <span class="tag" :class="`tag-${variant}`">
    <slot />
  </span>
</template>

<script>
export default {
  name: 'AppTag',
  props: {
    variant: {
      type: String,
      default: 'info',
      validator: (v) => ['batch', 'remote', 'info'].includes(v),
    },
  },
}
</script>
