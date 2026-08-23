<template>
  <div class="app-chrome-icons">
    <AppIconBtn title="系统设置" aria-label="系统设置" @click="openSettings">
      <el-icon :size="15"><Setting /></el-icon>
    </AppIconBtn>
    <AppIconBtn title="关于 FlashShell" aria-label="关于" @click="openAbout">
      <el-icon :size="15"><QuestionFilled /></el-icon>
    </AppIconBtn>
    <WindowControls />
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Setting, QuestionFilled } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'
import WindowControls from './WindowControls.vue'
import { AppIconBtn } from './ui'

export default {
  name: 'AppChromeIcons',
  components: { Setting, QuestionFilled, WindowControls, AppIconBtn },
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

    const openSettings = () => { App.OpenSystemSettings() }
    const openAbout = () => { App.OpenAbout() }

    let offShortcutsChanged = null

    onMounted(() => {
      loadShortcuts()
      offShortcutsChanged = EventsOn('shortcuts:changed', (data) => {
        shortcuts.value = mergeShortcuts(data)
      })
    })
    onUnmounted(() => {
      offShortcutsChanged?.()
      offShortcutsChanged = null
    })

    return {
      newWindowLabel,
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
  gap: 2px;
  flex-shrink: 0;
  height: 100%;
}
</style>
