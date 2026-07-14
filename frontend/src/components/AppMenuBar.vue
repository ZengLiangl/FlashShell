<template>
  <div class="app-menu-bar">
    <div class="menu-icons">
      <el-dropdown trigger="click" @command="onFileCommand">
        <button type="button" class="icon-btn" title="文件">
          <el-icon :size="16"><DocumentAdd /></el-icon>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="new-window">
              <span>新建窗口</span>
              <span class="menu-shortcut">{{ labelOf('newWindow') }}</span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <el-dropdown trigger="click" @command="onConfigCommand">
        <button type="button" class="icon-btn" title="配置文件">
          <el-icon :size="16"><FolderOpened /></el-icon>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <template v-if="configFiles.length">
              <el-dropdown-item
                v-for="file in configFiles"
                :key="file"
                :command="`switch:${file}`"
              >
                <span class="config-item">
                  <el-icon v-if="file === currentConfig" class="config-check"><Check /></el-icon>
                  <span>{{ basename(file) }}</span>
                </span>
              </el-dropdown-item>
            </template>
            <el-dropdown-item v-else disabled>无法加载配置文件</el-dropdown-item>
            <el-dropdown-item divided command="refresh">
              <span>刷新配置列表</span>
              <span class="menu-shortcut">{{ labelOf('refreshConfig') }}</span>
            </el-dropdown-item>
            <el-dropdown-item command="open-global">打开全局配置</el-dropdown-item>
            <el-dropdown-item command="open-current">打开当前配置</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

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
      <!-- <el-dropdown trigger="click" @command="onHelpCommand">
        <button type="button" class="icon-btn" title="帮助">
          <el-icon :size="16"><QuestionFilled /></el-icon>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="about">关于</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown> -->
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { Check, DocumentAdd, FolderOpened, Setting, QuestionFilled } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { mergeShortcuts, formatShortcut } from '../utils/shortcuts'

function basename(filePath) {
  if (!filePath) return ''
  const normalized = filePath.replace(/\\/g, '/')
  const idx = normalized.lastIndexOf('/')
  return idx >= 0 ? normalized.slice(idx + 1) : filePath
}

export default {
  name: 'AppMenuBar',
  components: { Check, DocumentAdd, FolderOpened, Setting, QuestionFilled },
  setup() {
    const configFiles = ref([])
    const currentConfig = ref('')
    const shortcuts = ref(mergeShortcuts())

    const labelOf = (id) => formatShortcut(shortcuts.value[id])

    const loadShortcuts = async () => {
      try {
        shortcuts.value = mergeShortcuts(await App.GetShortcutSettings())
      } catch {
        shortcuts.value = mergeShortcuts()
      }
    }

    const loadMenuData = async () => {
      try {
        const [files, current] = await Promise.all([
          App.GetConfigFiles(),
          App.GetCurrentConfigPath(),
        ])
        configFiles.value = files || []
        currentConfig.value = current || ''
      } catch (error) {
        console.error('加载菜单数据失败:', error)
        configFiles.value = []
        currentConfig.value = ''
      }
    }

    const openSettings = () => {
      App.OpenSystemSettings()
    }

    const onFileCommand = (cmd) => {
      if (cmd === 'new-window') App.NewWindow()
    }

    const onConfigCommand = (cmd) => {
      if (cmd === 'refresh') {
        App.RefreshConfigMenuWithEvent()
        return
      }
      if (cmd === 'open-global') {
        App.OpenGlobalConfigWithEvent()
        return
      }
      if (cmd === 'open-current') {
        App.OpenCurrentConfigWithEvent()
        return
      }
      if (typeof cmd === 'string' && cmd.startsWith('switch:')) {
        const file = cmd.slice('switch:'.length)
        if (file && file !== currentConfig.value) {
          App.SwitchConfigFileWithEvent(file)
        }
      }
    }

    const onHelpCommand = (cmd) => {
      if (cmd === 'about') App.OpenAbout()
    }

    onMounted(() => {
      loadMenuData()
      loadShortcuts()
      EventsOn('menu:refresh', loadMenuData)
      EventsOn('shortcuts:changed', (data) => {
        shortcuts.value = mergeShortcuts(data)
      })
    })

    onUnmounted(() => {
      EventsOff('menu:refresh')
      EventsOff('shortcuts:changed')
    })

    return {
      configFiles,
      currentConfig,
      basename,
      labelOf,
      openSettings,
      onFileCommand,
      onConfigCommand,
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

.menu-shortcut {
  margin-left: 24px;
  color: var(--app-text-muted);
  font-size: 12px;
}

.config-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.config-check {
  color: var(--app-accent-color);
  font-size: 14px;
}
</style>
