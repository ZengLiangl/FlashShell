import { createApp } from 'vue'
import { ElLoading } from 'element-plus'
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

const app = createApp(App)

app.use(ElLoading)
registerAppIcons(app)

app.mount('#app')
installPopperAutoClose()
