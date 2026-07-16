<template>
  <div class="app-menu-bar">
    <div class="menu-icons">
      <button
        type="button"
        class="icon-btn"
        :title="`新建窗口 ${labelOf('newWindow')}`"
        @click="newWindow"
      >
        <el-icon :size="16"><DocumentAdd /></el-icon>
      </button>

      <button
        type="button"
        class="icon-btn"
        title="系统设置"
        @click="openSettings"
      >
        <el-icon :size="16"><Setting /></el-icon>
      </button>

      <button type="button" class="icon-btn" title="帮助" @click="onHelpCommand('about')">
        <el-icon :size="16"><QuestionFilled /></el-icon>
      </button>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { DocumentAdd, Setting, QuestionFilled } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'

export default {
  name: 'AppMenuBar',
  components: { DocumentAdd, Setting, QuestionFilled },
  setup() {
    const shortcuts = ref(mergeShortcuts())

    const labelOf = (id) => formatShortcut(shortcuts.value[id])

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const newWindow = () => {
      App.NewWindow()
    }

    const openSettings = () => {
      App.OpenSystemSettings()
    }

    const onHelpCommand = (cmd) => {
      if (cmd === 'about') App.OpenAbout()
    }

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
      labelOf,
      newWindow,
      openSettings,
      onHelpCommand,
    }
  },
}
</script>

<style scoped>
.app-menu-bar {
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
  color: var(--app-text);
  height: 36px;
  display: flex;
  align-items: center;
  padding: 0 8px;
}

.menu-icons {
  display: flex;
  align-items: center;
  gap: 2px;
}

.icon-btn {
  width: 32px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--app-text-secondary, var(--app-text));
  cursor: pointer;
  padding: 0;
}

.icon-btn:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}
</style>
