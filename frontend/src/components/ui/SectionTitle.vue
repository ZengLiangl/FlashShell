<template>
  <div class="home-section-title">
    <slot name="icon" />
    <span>{{ title }}</span>
    <span v-if="count != null" class="count">{{ count }}</span>
    <span v-if="$slots.actions" class="section-actions">
      <slot name="actions" />
    </span>
  </div>
</template>

<script>
export default {
  name: 'SectionTitle',
  props: {
    title: { type: String, required: true },
    count: { type: [Number, String], default: null },
  },
}
</script>

<style scoped>
.section-actions {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}
</style>
