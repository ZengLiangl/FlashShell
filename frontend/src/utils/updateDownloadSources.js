/** 与后端 app/update.go buildDownloadSources 保持一致的默认列表（缓存旧结果无 sources 时兜底） */
export const DEFAULT_UPDATE_DOWNLOAD_SOURCES = [
  { label: 'GitHub' },
  { label: 'ghfast.top' },
  { label: 'ghproxy.net' },
  { label: 'gitclone.com' },
  { label: 'githubproxy.cc' },
  { label: 'github.abskoop.workers.dev' },
]

export function resolveUpdateDownloadSources(updateResult) {
  const list = updateResult?.downloadSources
  if (Array.isArray(list) && list.length) {
    return list
      .map((s) => ({ label: String(s?.label || '').trim() }))
      .filter((s) => s.label)
  }
  if (updateResult?.downloadURL) {
    return DEFAULT_UPDATE_DOWNLOAD_SOURCES.map((s) => ({ ...s }))
  }
  return []
}
