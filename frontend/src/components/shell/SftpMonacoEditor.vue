<template>
  <div ref="hostRef" class="sftp-monaco-host" />
</template>

<script>
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { setupMonacoEnvironment, monacoLanguageFromPath } from './monacoEnv.js'

export default {
  name: 'SftpMonacoEditor',
  props: {
    modelValue: { type: String, default: '' },
    path: { type: String, default: '' },
    readOnly: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const hostRef = ref(null)
    let editor = null
    let monaco = null
    let applying = false

    const themeFromDom = () => {
      const root = document.documentElement
      const dark =
        root.classList.contains('dark') ||
        root.getAttribute('data-theme') === 'dark' ||
        window.matchMedia?.('(prefers-color-scheme: dark)')?.matches
      return dark ? 'vs-dark' : 'vs'
    }

    const syncLanguage = () => {
      if (!editor || !monaco) return
      const model = editor.getModel()
      if (!model) return
      monaco.editor.setModelLanguage(model, monacoLanguageFromPath(props.path))
    }

    const mountEditor = async () => {
      if (!hostRef.value || editor) return
      setupMonacoEnvironment()
      monaco = await import('monaco-editor')
      editor = monaco.editor.create(hostRef.value, {
        value: props.modelValue || '',
        language: monacoLanguageFromPath(props.path),
        theme: themeFromDom(),
        readOnly: props.readOnly,
        automaticLayout: true,
        minimap: { enabled: false },
        fontSize: 13,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        tabSize: 2,
        renderWhitespace: 'selection',
      })
      editor.onDidChangeModelContent(() => {
        if (applying) return
        emit('update:modelValue', editor.getValue())
      })
    }

    onMounted(() => {
      nextTick(() => {
        void mountEditor()
      })
    })

    onBeforeUnmount(() => {
      if (editor) {
        editor.dispose()
        editor = null
      }
    })

    watch(
      () => props.modelValue,
      (v) => {
        if (!editor) return
        const next = v == null ? '' : String(v)
        if (editor.getValue() === next) return
        applying = true
        editor.setValue(next)
        applying = false
      },
    )

    watch(
      () => props.path,
      () => syncLanguage(),
    )

    watch(
      () => props.readOnly,
      (v) => {
        editor?.updateOptions({ readOnly: !!v })
      },
    )

    return { hostRef }
  },
}
</script>

<style scoped>
.sftp-monaco-host {
  width: 100%;
  height: min(62vh, 560px);
  min-height: 320px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
}
</style>
