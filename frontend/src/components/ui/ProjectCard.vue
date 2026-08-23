<template>
  <button type="button" class="project-card" @click="$emit('click', $event)">
    <div class="pc-top">
      <span class="pc-icon">
        <slot name="icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <rect x="4" y="3" width="16" height="18" rx="2" />
            <path d="M8 8h8M8 12h8M8 16h5" />
          </svg>
        </slot>
      </span>
      <div>
        <div class="pc-name">{{ name }}</div>
        <div v-if="description" class="pc-desc">{{ description }}</div>
      </div>
    </div>
    <div class="pc-foot">
      <span v-if="meta" class="pc-meta">{{ meta }}</span>
      <span class="pc-run btn btn-sm btn-soft">
        进入
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" style="width:11px;height:11px" aria-hidden="true">
          <path d="M9 6l6 6-6 6" />
        </svg>
      </span>
    </div>
  </button>
</template>

<script>
export default {
  name: 'ProjectCard',
  props: {
    name: { type: String, required: true },
    description: { type: String, default: '' },
    meta: { type: String, default: '' },
  },
  emits: ['click'],
}
</script>
