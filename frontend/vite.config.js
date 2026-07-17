import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
    plugins: [vue()],
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
        port: 3000,
    },
})
