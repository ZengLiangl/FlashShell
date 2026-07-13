<template>
  <div class="app-menu-bar">
    <el-menu
      :key="menuKey"
      mode="horizontal"
      :ellipsis="false"
      class="menu-inner"
      @select="onMenuSelect"
    >
      <el-sub-menu index="file">
        <template #title>文件</template>
        <el-menu-item index="file-new" @click="onNewWindow">
          新建窗口
          <span class="menu-shortcut">Ctrl+N</span>
        </el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="settings">
        <template #title>设置</template>
        <el-menu-item index="settings-machine" @click="App.OpenMachineConfig()">
          机器配置
          <span class="menu-shortcut">Ctrl+M</span>
        </el-menu-item>
        <el-menu-item index="settings-env" @click="App.OpenWorkPathConfig()">
          环境变量
          <span class="menu-shortcut">Ctrl+E</span>
        </el-menu-item>
        <el-menu-item divided index="settings-config-editor" @click="App.OpenConfigEditor()">
          业务配置编辑
          <span class="menu-shortcut">Ctrl+,</span>
        </el-menu-item>
        <el-menu-item index="settings-system" @click="App.OpenSystemSettings()">系统设置</el-menu-item>
        <el-menu-item index="settings-history" @click="App.OpenExecutionHistory()">执行历史</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="configs">
        <template #title>配置文件</template>
        <template v-if="configFiles.length">
          <el-menu-item
            v-for="file in configFiles"
            :key="file"
            :index="`config-${file}`"
            @click="switchConfig(file)"
          >
            <span class="config-item">
              <el-icon v-if="file === currentConfig" class="config-check"><Check /></el-icon>
              <span>{{ basename(file) }}</span>
            </span>
          </el-menu-item>
        </template>
        <el-menu-item v-else index="config-empty" disabled>无法加载配置文件</el-menu-item>
        <el-menu-item divided index="config-refresh" @click="refreshConfigList">
          刷新配置列表
          <span class="menu-shortcut">Ctrl+R</span>
        </el-menu-item>
        <el-menu-item index="config-global" @click="App.OpenGlobalConfigWithEvent()">打开全局配置</el-menu-item>
        <el-menu-item index="config-current" @click="App.OpenCurrentConfigWithEvent()">打开当前配置</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="help">
        <template #title>帮助</template>
        <el-menu-item index="help-about" @click="App.OpenAbout()">关于</el-menu-item>
      </el-sub-menu>
    </el-menu>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Check } from '@element-plus/icons-vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

function basename(filePath) {
  if (!filePath) return ''
  const normalized = filePath.replace(/\\/g, '/')
  const idx = normalized.lastIndexOf('/')
  return idx >= 0 ? normalized.slice(idx + 1) : filePath
}

export default {
  name: 'AppMenuBar',
  components: { Check },
  setup() {
    const configFiles = ref([])
    const currentConfig = ref('')
    const menuKey = ref(0)

    const clearMenuHighlight = () => {
      nextTick(() => {
        menuKey.value += 1
      })
    }

    const onMenuSelect = () => {
      clearMenuHighlight()
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

    const onNewWindow = () => App.NewWindow()
    const refreshConfigList = () => App.RefreshConfigMenuWithEvent()
    const switchConfig = (file) => {
      if (file && file !== currentConfig.value) {
        App.SwitchConfigFileWithEvent(file)
      }
    }

    onMounted(() => {
      loadMenuData()
      EventsOn('menu:refresh', loadMenuData)
    })

    onUnmounted(() => {
      EventsOff('menu:refresh')
    })

    return {
      App,
      configFiles,
      currentConfig,
      menuKey,
      basename,
      onNewWindow,
      refreshConfigList,
      switchConfig,
      onMenuSelect,
    }
  },
}
</script>

<style scoped>
.app-menu-bar {
  flex-shrink: 0;
  border-bottom: 1px solid var(--app-border);
  background: var(--app-panel-bg);
}

.menu-inner {
  border-bottom: none;
  background: transparent;
  height: 36px;
}

.menu-inner :deep(.el-menu-item),
.menu-inner :deep(.el-sub-menu__title) {
  height: 36px;
  line-height: 36px;
  color: var(--app-text);
}

.menu-inner :deep(.el-menu-item:hover),
.menu-inner :deep(.el-sub-menu__title:hover) {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.menu-inner :deep(.el-menu-item.is-active) {
  color: var(--app-text);
  background: transparent;
}

.menu-inner :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: var(--app-text);
  border-bottom-color: transparent;
}

.menu-shortcut {
  float: right;
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
