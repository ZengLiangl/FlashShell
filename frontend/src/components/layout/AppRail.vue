<template>
  <nav class="rail" aria-label="主导航">
    <button
      type="button"
      class="nav-item"
      :class="{ active: activeView === 'home' }"
      title="首页"
      aria-label="首页"
      @click="$emit('change-view', 'home')"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M3 10.5L12 3l9 7.5" /><path d="M5 9.5V21h14V9.5" />
      </svg>
    </button>
    <button
      v-if="hasProjects"
      type="button"
      class="nav-item"
      :class="{ active: activeView === 'task' }"
      title="任务"
      aria-label="任务"
      @click="$emit('change-view', 'task')"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <rect x="4" y="3" width="16" height="18" rx="2" /><path d="M8 8h8M8 12h8M8 16h5" />
      </svg>
    </button>
    <button
      type="button"
      class="nav-item"
      :class="{ active: activeView === 'shell' }"
      title="终端"
      aria-label="终端"
      @click="$emit('change-view', 'shell')"
    >
      <span v-if="sessionBadge > 0" class="ncount">{{ sessionBadge }}</span>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M4 17l6-6-6-6M12 19h8" />
      </svg>
    </button>

    <div class="rail-gap" />

    <span class="rail-tip">FlashShell</span>
    <div class="rail-sep" aria-hidden="true" />

    <button type="button" class="nav-item" title="系统设置" aria-label="系统设置" @click="$emit('open-settings')">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1" />
      </svg>
    </button>
    <button
      type="button"
      class="rail-mode"
      title="切换界面模式"
      aria-label="切换界面模式"
      :aria-pressed="isDark ? 'true' : 'false'"
      @click="toggleTheme"
    >
      <svg class="i-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
      </svg>
      <svg class="i-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M21 12.8A9 9 0 1111.2 3a7 7 0 009.8 9.8z" />
      </svg>
    </button>
  </nav>
</template>

<script>
import { computed } from 'vue'
import { useTheme } from '../../composables/useTheme'

export default {
  name: 'AppRail',
  props: {
    activeView: {
      type: String,
      default: 'home',
      validator: (v) => ['home', 'task', 'shell'].includes(v),
    },
    openSessionCount: { type: Number, default: 0 },
    connectedCount: { type: Number, default: 0 },
    hasProjects: { type: Boolean, default: false },
  },
  emits: ['change-view', 'open-settings'],
  setup(props) {
    const { isDark, themeMode, saveTheme } = useTheme()

    const sessionBadge = computed(() => props.openSessionCount || props.connectedCount || 0)

    const toggleTheme = () => {
      const next = isDark.value ? 'light' : 'dark'
      if (themeMode.value === 'system') {
        saveTheme({ mode: next })
        return
      }
      saveTheme({ mode: next })
    }

    return { isDark, sessionBadge, toggleTheme }
  },
}
</script>
