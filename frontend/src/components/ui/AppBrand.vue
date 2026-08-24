<template>
  <div class="brand app-brand">
    <span class="brand-mark" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="currentColor">
        <path d="M13.2 2 3 14h6.5l-1 8L20 10h-6.5l-.3-8z" />
      </svg>
    </span>
    FlashShell
  </div>
</template>

<script>
export default {
  name: 'AppBrand',
}
</script>

<style scoped>
.app-brand {
  flex-shrink: 0;
  -webkit-app-region: drag;
}
</style>
