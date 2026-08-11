import { ElMessageBox } from 'element-plus'

const VAR_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}/g
const STORAGE_KEY = 'flashdock.snippet.vars.v1'

const readStore = () => {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}') || {}
  } catch {
    return {}
  }
}

const writeStore = (store) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store || {}))
  } catch {
    // ignore
  }
}

export function extractSnippetVariables(command) {
  const vars = new Set()
  if (!command) return []
  let m
  const re = /\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}/g
  while ((m = re.exec(command))) vars.add(m[1])
  return [...vars]
}

export function expandSnippetVariables(command, values = {}) {
  if (!command) return ''
  return command.replace(VAR_RE, (_, name) => {
    const v = values[name]
    return v != null ? String(v) : `{{${name}}}`
  })
}

export function getRememberedSnippetVars(snippetId) {
  if (!snippetId) return {}
  const store = readStore()
  return { ...(store[snippetId] || {}) }
}

export function rememberSnippetVars(snippetId, values) {
  if (!snippetId || !values) return
  const store = readStore()
  store[snippetId] = { ...(store[snippetId] || {}), ...values }
  writeStore(store)
}

/**
 * 解析片段命令：对 {{变量}} 提示一次并记住；prompt=false 时仅用已记住值，缺失则返回 null
 */
export async function resolveSnippetCommand(snippet, { prompt = true } = {}) {
  const command = snippet?.command || ''
  const vars = extractSnippetVariables(command)
  if (!vars.length) return command

  const remembered = getRememberedSnippetVars(snippet?.id)
  const values = { ...remembered }
  for (const name of vars) {
    if (values[name] != null && String(values[name]).trim() !== '') continue
    if (!prompt) return null
    try {
      const { value } = await ElMessageBox.prompt(`请输入变量「${name}」`, snippet?.name || '代码片段', {
        inputValue: values[name] || '',
        confirmButtonText: '确定',
        cancelButtonText: '跳过',
      })
      if (value != null) values[name] = value
    } catch {
      // 用户取消：保留占位符
    }
  }
  rememberSnippetVars(snippet?.id, values)
  const expanded = expandSnippetVariables(command, values)
  if (/\{\{[a-zA-Z_][a-zA-Z0-9_]*\}\}/.test(expanded)) return null
  return expanded
}
