import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './styles/theme.css'
import './styles/machine-list.css'
import App from './App.vue'
import { installPopperAutoClose } from './utils/popperAutoClose'
import { registerAppIcons } from './utils/registerIcons'

const app = createApp(App)

app.use(ElementPlus)
registerAppIcons(app)

app.mount('#app')
installPopperAutoClose()
