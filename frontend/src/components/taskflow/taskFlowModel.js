/** 任务流水线配置：空节点、步骤规范化、路径级写回 */

export function emptyProject() {
  return { name: '', description: '', workdir: '', subprojects: [] }
}

export function emptySubProject() {
  return { name: '', description: '', workdir: '', commands: [] }
}

export function emptyCommand(template = 'batch') {
  if (template === 'remote') {
    return {
      name: '远程部署',
      description: '',
      type: 'remote',
      machine: '',
      workdir: '',
      steps: [],
    }
  }
  return {
    name: '本机构建',
    description: '',
    type: 'batch',
    machine: '',
    workdir: '',
    steps: [],
  }
}

export function emptyStep(kind = 'shell') {
  const base = { command: '', onFail: 'abort', retry: 0, kind }
  switch (kind) {
    case 'upload':
      return { ...base, command: 'upload ', kind: 'upload' }
    case 'chdir':
      return { ...base, command: 'chdir ', kind: 'chdir' }
    case 'targz':
      return { ...base, command: 'targz ', kind: 'targz' }
    default:
      return { ...base, kind: 'shell' }
  }
}

export function deepClone(obj) {
  return JSON.parse(JSON.stringify(obj ?? null))
}

export function normalizeStep(s) {
  if (typeof s === 'string') {
    return {
      command: s,
      onFail: 'abort',
      retry: 0,
      kind: detectStepKind(s),
    }
  }
  const command = s.command || s.cmd || ''
  return {
    command,
    onFail: s.onFail || s.on_fail || 'abort',
    retry: s.retry || 0,
    kind: s.kind || detectStepKind(command),
  }
}

export function detectStepKind(command = '') {
  const t = String(command).trim()
  if (t.startsWith('upload ')) return 'upload'
  if (t.startsWith('chdir ')) return 'chdir'
  if (t.startsWith('targz ')) return 'targz'
  return 'shell'
}

export function normalizeCommand(cmd) {
  const next = {
    name: cmd.name || '',
    description: cmd.description || '',
    type: cmd.type === 'remote' ? 'remote' : 'batch',
    machine: cmd.machine || '',
    workdir: cmd.workdir || '',
    steps: (cmd.steps || []).map(normalizeStep),
  }
  return next
}

export function normalizeRoot(config) {
  const root = deepClone(config) || { projects: [], machines: [] }
  if (!Array.isArray(root.projects)) root.projects = []
  if (!Array.isArray(root.machines)) root.machines = []
  root.projects.forEach((p) => {
    if (!Array.isArray(p.subprojects)) p.subprojects = []
    p.subprojects.forEach((sp) => {
      if (!Array.isArray(sp.commands)) sp.commands = []
      sp.commands = sp.commands.map(normalizeCommand)
    })
  })
  return root
}

/**
 * 保存给 Wails/Go：steps 使用 json tag（command/onFail/retry）。
 * YAML 简写由后端 StepList.MarshalYAML 在落盘时处理。
 */
export function serializeStepsForSave(steps) {
  return (steps || []).map((s) => {
    const command = (s.command || '').trim()
    const onFail = s.onFail || 'abort'
    const retry = Number(s.retry) || 0
    const out = { command }
    if (onFail && onFail !== 'abort') out.onFail = onFail
    if (retry > 0) out.retry = retry
    return out
  })
}

export function serializeRootForSave(root) {
  const payload = deepClone(root)
  payload.projects?.forEach((p) => {
    p.subprojects?.forEach((sp) => {
      sp.commands?.forEach((cmd) => {
        cmd.steps = serializeStepsForSave(cmd.steps)
      })
    })
  })
  return payload
}

export function pathKey(path) {
  if (!path) return ''
  const { p, s, c, st } = path
  if (st != null) return `p${p}-s${s}-c${c}-st${st}`
  if (c != null) return `p${p}-s${s}-c${c}`
  if (s != null) return `p${p}-s${s}`
  if (p != null) return `p${p}`
  return ''
}

export function samePath(a, b) {
  if (!a || !b) return false
  return pathKey(a) === pathKey(b)
}

export function getProject(root, p) {
  return root?.projects?.[p] ?? null
}

export function getSubProject(root, p, s) {
  return getProject(root, p)?.subprojects?.[s] ?? null
}

export function getCommand(root, p, s, c) {
  return getSubProject(root, p, s)?.commands?.[c] ?? null
}

export function getStep(root, p, s, c, st) {
  return getCommand(root, p, s, c)?.steps?.[st] ?? null
}

/** 按路径读取节点深拷贝草稿 */
export function cloneNodeByPath(root, path) {
  if (!path || path.p == null) return null
  if (path.st != null) {
    const step = getStep(root, path.p, path.s, path.c, path.st)
    return step ? deepClone(step) : null
  }
  if (path.c != null) {
    const cmd = getCommand(root, path.p, path.s, path.c)
    return cmd ? deepClone({ ...cmd, steps: undefined }) : null
  }
  if (path.s != null) {
    const sub = getSubProject(root, path.p, path.s)
    return sub
      ? deepClone({
          name: sub.name,
          description: sub.description,
          workdir: sub.workdir,
        })
      : null
  }
  const project = getProject(root, path.p)
  return project
    ? deepClone({
        name: project.name,
        description: project.description,
        workdir: project.workdir,
      })
    : null
}

/**
 * 仅写回选中路径对应字段，不替换兄弟节点引用。
 * command 草稿不含 steps；step 草稿不含兄弟。
 */
export function commitByPath(root, path, draft) {
  if (!root || !path || !draft) return false
  if (path.st != null) {
    const cmd = getCommand(root, path.p, path.s, path.c)
    if (!cmd || !cmd.steps?.[path.st]) return false
    const target = cmd.steps[path.st]
    target.command = draft.command ?? ''
    target.onFail = draft.onFail || 'abort'
    target.retry = draft.retry || 0
    target.kind = draft.kind || detectStepKind(target.command)
    return true
  }
  if (path.c != null) {
    const cmd = getCommand(root, path.p, path.s, path.c)
    if (!cmd) return false
    cmd.name = draft.name ?? ''
    cmd.description = draft.description ?? ''
    cmd.type = draft.type === 'remote' ? 'remote' : 'batch'
    cmd.machine = draft.machine ?? ''
    cmd.workdir = draft.workdir ?? ''
    return true
  }
  if (path.s != null) {
    const sub = getSubProject(root, path.p, path.s)
    if (!sub) return false
    sub.name = draft.name ?? ''
    sub.description = draft.description ?? ''
    sub.workdir = draft.workdir ?? ''
    return true
  }
  if (path.p != null) {
    const project = getProject(root, path.p)
    if (!project) return false
    project.name = draft.name ?? ''
    project.description = draft.description ?? ''
    project.workdir = draft.workdir ?? ''
    return true
  }
  return false
}

export function stepSummary(step) {
  const cmd = (step?.command || '').trim()
  if (!cmd) return '(空步骤)'
  if (cmd.length > 42) return `${cmd.slice(0, 40)}…`
  return cmd
}

export function commandTypeLabel(type) {
  return type === 'remote' ? '远程' : '本机'
}

export function parseUploadPaths(command = '') {
  const parts = String(command).trim().split(/\s+/)
  if (parts[0] !== 'upload') return { local: '', remote: '' }
  return { local: parts[1] || '', remote: parts[2] || '' }
}

export function buildUploadCommand(local, remote) {
  return `upload ${local || ''} ${remote || ''}`.trim()
}

export function parseChdirPath(command = '') {
  const t = String(command).trim()
  if (!t.startsWith('chdir ')) return ''
  return t.slice(6).trim()
}

export function buildChdirCommand(path) {
  return `chdir ${path || ''}`.trim()
}

export function parseTargzPaths(command = '') {
  const parts = String(command).trim().split(/\s+/)
  if (parts[0] !== 'targz') return { src: '', dest: '' }
  return { src: parts[1] || '', dest: parts[2] || '' }
}

export function buildTargzCommand(src, dest) {
  return `targz ${src || ''} ${dest || ''}`.trim()
}

/** 将数组元素从 from 移到 to（原地修改） */
export function moveArrayItem(arr, from, to) {
  if (!Array.isArray(arr) || from === to) return false
  if (from < 0 || to < 0 || from >= arr.length || to >= arr.length) return false
  const [item] = arr.splice(from, 1)
  arr.splice(to, 0, item)
  return true
}

/** 单列表重排后，把选中下标映射到新位置 */
export function remapIndexAfterMove(index, from, to) {
  if (index == null || from === to) return index
  if (index === from) return to
  if (from < to) {
    if (index > from && index <= to) return index - 1
  } else if (index >= to && index < from) {
    return index + 1
  }
  return index
}

/**
 * 移动步骤（可跨命令）。返回落点 { c, st }；失败返回 null。
 * toSt 为插入目标下标（同命令内按 moveArrayItem 语义）。
 */
export function moveStep(commands, fromC, fromSt, toC, toSt) {
  if (!Array.isArray(commands)) return null
  const fromCmd = commands[fromC]
  const toCmd = commands[toC]
  if (!fromCmd || !toCmd) return null
  if (!Array.isArray(fromCmd.steps)) fromCmd.steps = []
  if (!Array.isArray(toCmd.steps)) toCmd.steps = []
  if (fromSt < 0 || fromSt >= fromCmd.steps.length) return null

  if (fromC === toC) {
    let target = toSt
    if (target >= fromCmd.steps.length) target = fromCmd.steps.length - 1
    if (target < 0) return null
    if (!moveArrayItem(fromCmd.steps, fromSt, target)) return null
    return { c: toC, st: target }
  }

  const [item] = fromCmd.steps.splice(fromSt, 1)
  const insertAt = Math.max(0, Math.min(toSt, toCmd.steps.length))
  toCmd.steps.splice(insertAt, 0, item)
  return { c: toC, st: insertAt }
}
