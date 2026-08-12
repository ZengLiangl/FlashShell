import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            resolvers: [ElementPlusResolver({ importStyle: 'css' })],
            dts: 'src/auto-imports.d.ts',
        }),
        Components({
            resolvers: [ElementPlusResolver({ importStyle: 'css' })],
            dts: 'src/components.d.ts',
        }),
    ],
    build: {
        outDir: 'dist',
        assetsDir: 'assets',
        rollupOptions: {
            external: [],
            output: {
                manualChunks(id) {
                    if (id.includes('node_modules/xterm')) {
                        return 'xterm'
                    }
                    if (id.includes('node_modules/monaco-editor')) {
                        return 'monaco'
                    }
                    if (id.includes('node_modules/element-plus')) {
                        return 'element-plus'
                    }
                    if (id.includes('/views/ShellWorkspace.vue') || id.includes('/components/shell/')) {
                        return 'shell'
                    }
                },
            },
        },
    },
    server: {
        // 必须与 wails.json 的 frontend:dev:serverUrl 端口一致
        port: 3000,
        strictPort: true,
    },
})
