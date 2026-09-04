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

    <button
      v-if="showAudit"
      type="button"
      class="nav-item"
      :class="{ active: activeView === 'audit' }"
      title="审计 / 敏感库"
      aria-label="审计 / 敏感库"
      @click="$emit('change-view', 'audit')"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
      </svg>
    </button>

    <div class="rail-gap" />

    <span class="rail-tip">FlashShell</span>
    <div class="rail-sep" aria-hidden="true" />

    <button type="button" class="nav-item" title="系统设置" aria-label="系统设置" @click="$emit('open-settings')">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
        <circle cx="12" cy="12" r="3" />
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
      validator: (v) => ['home', 'task', 'shell', 'audit'].includes(v),
    },
    openSessionCount: { type: Number, default: 0 },
    connectedCount: { type: Number, default: 0 },
    hasProjects: { type: Boolean, default: false },
    showAudit: { type: Boolean, default: false },
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
