import * as App from '../../wailsjs/go/app/App'
import { expandSendString } from './keymaps'
import { resolveSnippetCommand } from './snippetVariables'

const ranForSession = new Set()

export function resetOnConnectSnippets(sessionId = '') {
  if (!sessionId) {
    ranForSession.clear()
    return
  }
  ranForSession.delete(sessionId)
}

export function snippetMatchesSession(snippet, configName) {
  if (!snippet?.onConnect) return false
  const scope = String(snippet.scope || 'global').trim() || 'global'
  if (scope === 'global') return true
  return scope === String(configName || '').trim()
}

export async function runOnConnectSnippets(session, snippets) {
  const sessionId = session?.machineName
  if (!sessionId || !session?.connected || session?.connecting) return
  if (ranForSession.has(sessionId)) return
  ranForSession.add(sessionId)

  const configName = session.configName || sessionId
  const list = (snippets || []).filter((s) => snippetMatchesSession(s, configName))
  if (!list.length) return

  for (const snippet of list) {
    try {
      const resolved = await resolveSnippetCommand(snippet, { prompt: false })
      if (!resolved) continue
      let text = expandSendString(resolved)
      if (snippet.execute && text && !/[\r\n]$/.test(text)) text += '\n'
      if (!text) continue
      await App.SendShellInput(sessionId, text)
    } catch {
      // 单条失败不阻断其余片段
    }
  }
}
