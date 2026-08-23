import { ref, onMounted, onUnmounted } from 'vue'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

function basename(filePath) {
  if (!filePath) return ''
  const normalized = String(filePath).replace(/\\/g, '/')
  const idx = normalized.lastIndexOf('/')
  return idx >= 0 ? normalized.slice(idx + 1) : filePath
}

export function useConfigFileMenu({ onEditPipeline, onReload } = {}) {
  const configFiles = ref([])
  const currentConfig = ref('')
  const configMenuOpen = ref(false)

  const loadConfigMenu = async () => {
    try {
      const [files, current] = await Promise.all([
        App.GetConfigFiles(),
        App.GetCurrentConfigPath(),
      ])
      configFiles.value = files || []
      currentConfig.value = current || ''
    } catch {
      configFiles.value = []
      currentConfig.value = ''
    }
  }

  const closeConfigMenu = () => {
    configMenuOpen.value = false
  }

  const toggleConfigMenu = (e) => {
    e?.stopPropagation?.()
    configMenuOpen.value = !configMenuOpen.value
  }

  const onConfigCommand = (cmd) => {
    closeConfigMenu()
    if (cmd === 'edit-pipeline') {
      onEditPipeline?.()
      return
    }
    if (cmd === 'reload') {
      onReload?.()
      return
    }
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

  let offConfigChanged = null

  onMounted(() => {
    loadConfigMenu()
    offConfigChanged = EventsOn('config:changed', loadConfigMenu)
  })

  onUnmounted(() => {
    offConfigChanged?.()
    offConfigChanged = null
  })

  return {
    configFiles,
    currentConfig,
    configMenuOpen,
    basename,
    loadConfigMenu,
    toggleConfigMenu,
    closeConfigMenu,
    onConfigCommand,
  }
}
