<template>
  <div class="app-chrome-icons">
    <!-- 新建窗口：暂隐藏，快捷键仍可用 -->
    <!--
    <button
      type="button"
      class="chrome-icon-btn"
      :title="`新建窗口 ${newWindowLabel}`"
      @click="newWindow"
    >
      <el-icon :size="14"><DocumentAdd /></el-icon>
    </button>
    -->
    <button
      type="button"
      class="chrome-icon-btn"
      title="系统设置"
      @click="openSettings"
    >
      <el-icon :size="14"><Setting /></el-icon>
    </button>
    <button
      type="button"
      class="chrome-icon-btn"
      title="帮助"
      @click="openAbout"
    >
      <el-icon :size="14"><QuestionFilled /></el-icon>
    </button>
    <WindowControls />
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Setting, QuestionFilled } from '@element-plus/icons-vue'
// import { DocumentAdd } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'
import WindowControls from './WindowControls.vue'

export default {
  name: 'AppChromeIcons',
  components: { Setting, QuestionFilled, WindowControls },
  // components: { DocumentAdd, Setting, QuestionFilled, WindowControls },
  setup() {
    const shortcuts = ref(mergeShortcuts())
    const newWindowLabel = computed(() => formatShortcut(shortcuts.value.newWindow))

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const newWindow = () => { App.NewWindow() }
    const openSettings = () => { App.OpenSystemSettings() }
    const openAbout = () => { App.OpenAbout() }

    onMounted(() => {
      loadShortcuts()
      EventsOn('shortcuts:changed', (data) => {
        shortcuts.value = mergeShortcuts(data)
      })
    })
    onUnmounted(() => {
      EventsOff('shortcuts:changed')
    })

    return {
      newWindowLabel,
      newWindow,
      openSettings,
      openAbout,
    }
  },
}
</script>

<style scoped>
.app-chrome-icons {
  display: inline-flex;
  align-items: center;
  gap: 1px;
  flex-shrink: 0;
  height: 100%;
}

.chrome-icon-btn {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--app-text-secondary, var(--app-text-muted, var(--app-text)));
  cursor: pointer;
  padding: 0;
  transition: background 0.15s ease, color 0.15s ease;
}

.chrome-icon-btn:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}
</style>
