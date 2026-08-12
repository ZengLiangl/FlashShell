import { createApp } from 'vue'
import { ElLoading } from 'element-plus'
import ElTooltip from 'element-plus/es/components/tooltip/index.mjs'
import 'element-plus/theme-chalk/dark/css-vars.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/overlay/style/css'
import 'element-plus/es/components/loading/style/css'
import './styles/theme.css'
import './styles/machine-list.css'
import './styles/scrollbar-hover.css'
import App from './App.vue'
import { installPopperAutoClose } from './utils/popperAutoClose'
import { registerAppIcons } from './utils/registerIcons'
import { isMacPlatform, isWindowsPlatform } from './utils/platform'

// 按钮点击获得焦点后，Enter/Space 会再次 toggle tooltip；图标按钮场景关闭该行为
if (ElTooltip?.props?.triggerKeys) {
  ElTooltip.props.triggerKeys.default = () => []
}

// macOS 隐藏标题栏 / Windows 无边框：顶栏让位系统按钮并启用拖拽
if (isMacPlatform()) {
  document.documentElement.classList.add('is-mac-hidden-titlebar')
}
if (isWindowsPlatform()) {
  document.documentElement.classList.add('is-windows-frameless')
}

const app = createApp(App)

app.use(ElLoading)
registerAppIcons(app)

app.mount('#app')
installPopperAutoClose()
