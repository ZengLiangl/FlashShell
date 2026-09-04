<template>
  <div class="sc-panel">
    <p class="sc-tip">
      正式服务凭据台账。列表与操作均不提供明文复制；请用「写远端」或 MCP
      <code>write_from_vault</code> / <code>sftp_write</code> 的
      <code v-pre>{{vault:id.field}}</code>
      在不暴露明文的情况下写入服务器。可添加与编辑备注。
    </p>
    <div class="sc-filters">
      <el-select
        v-model="serverFilter"
        clearable
        filterable
        size="small"
        style="width: 200px"
        placeholder="全部 / 按服务器"
        @change="reload"
        @clear="reload"
      >
        <el-option label="共用（未绑定）" value="__shared__" />
        <el-option v-for="a in serverAliases" :key="a" :label="a" :value="a" />
      </el-select>
      <el-button size="small" @click="reload">查询</el-button>
    </div>
    <el-table
      :data="rows"
      size="small"
      max-height="420"
      empty-text="暂无服务凭据"
      row-key="id"
      :row-class-name="rowClass"
    >
      <el-table-column prop="id" label="ID" width="110" />
      <el-table-column prop="label" label="标签" min-width="120" show-overflow-tooltip />
      <el-table-column prop="kind" label="类型" width="110" show-overflow-tooltip />
      <el-table-column prop="serverAlias" label="服务器" width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ row.serverAlias || '共用' }}</template>
      </el-table-column>
      <el-table-column prop="notes" label="备注" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.notes || '—' }}</template>
      </el-table-column>
      <el-table-column prop="secretFields" label="敏感字段" min-width="100" show-overflow-tooltip>
        <template #default="{ row }">{{ (row.secretFields || []).join(', ') || '—' }}</template>
      </el-table-column>
      <el-table-column prop="fromSensitive" label="来源" width="100" show-overflow-tooltip>
        <template #default="{ row }">
          <el-button
            v-if="row.fromSensitive"
            text
            size="small"
            type="primary"
            @click="$emit('jump-sensitive', row.fromSensitive)"
          >{{ row.fromSensitive }}</el-button>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建" width="150" show-overflow-tooltip />
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <div class="sc-ops">
            <el-button text size="small" @click="openNotes(row)">备注</el-button>
            <el-button
              text
              size="small"
              :disabled="!(row.secretFields && row.secretFields.length)"
              @click="openWrite(row)"
            >写远端</el-button>
            <el-button text size="small" type="danger" @click="onDelete(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="addOpen" title="手动添加服务凭据" width="480px" append-to-body>
      <div class="sc-form">
        <div class="row"><span>服务器</span>
          <el-select
            v-model="addForm.server"
            clearable
            filterable
            size="small"
            style="width: 100%"
            placeholder="不选 = 共用凭据"
          >
            <el-option v-for="a in serverAliases" :key="a" :label="a" :value="a" />
          </el-select>
        </div>
        <p class="sc-tip" style="margin: -4px 0 0">可不选服务器，表示各机共用的凭据。</p>
        <div class="row"><span>标签</span><el-input v-model="addForm.label" size="small" placeholder="MySQL / stripe_key" /></div>
        <div class="row"><span>类型</span>
          <el-select v-model="addForm.kind" size="small" filterable allow-create style="width: 100%">
            <el-option label="credential" value="credential" />
            <el-option label="token" value="token" />
            <el-option label="mysql_conn" value="mysql_conn" />
            <el-option label="redis_conn" value="redis_conn" />
            <el-option label="postgres_conn" value="postgres_conn" />
          </el-select>
        </div>
        <div class="row"><span>字段名</span><el-input v-model="addForm.field" size="small" placeholder="password / token" /></div>
        <div class="row"><span>敏感值</span><el-input v-model="addForm.value" type="password" show-password size="small" placeholder="不会回显到列表，且不可再复制" /></div>
        <div class="row"><span>备注</span>
          <el-input v-model="addForm.notes" type="textarea" :rows="2" size="small" placeholder="用途、到期提醒等" />
        </div>
      </div>
      <template #footer>
        <el-button size="small" @click="addOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="doAdd">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="notesOpen" title="编辑备注" width="440px" append-to-body>
      <div class="sc-form">
        <div class="row"><span>凭据 ID</span><el-input v-model="notesForm.id" disabled size="small" /></div>
        <div class="row"><span>标签</span><el-input v-model="notesForm.label" disabled size="small" /></div>
        <div class="row"><span>备注</span>
          <el-input v-model="notesForm.notes" type="textarea" :rows="3" size="small" placeholder="用途、到期提醒等" />
        </div>
      </div>
      <template #footer>
        <el-button size="small" @click="notesOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="doNotes">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="writeOpen" title="写到远端" width="480px" append-to-body>
      <div class="sc-form">
        <p class="sc-tip">本机解密后经 SFTP 写入；对话与界面不展示、不复制明文。</p>
        <div class="row"><span>凭据 ID</span><el-input v-model="writeForm.vaultId" disabled size="small" /></div>
        <div class="row"><span>服务器</span>
          <el-select
            v-model="writeForm.server"
            filterable
            size="small"
            style="width: 100%"
            placeholder="选择目标服务器"
          >
            <el-option v-for="a in serverAliases" :key="a" :label="a" :value="a" />
          </el-select>
        </div>
        <div class="row"><span>字段</span>
          <el-select v-model="writeForm.field" size="small" style="width: 100%">
            <el-option v-for="f in writeForm.fields" :key="f" :label="f" :value="f" />
          </el-select>
        </div>
        <div class="row"><span>远端路径</span><el-input v-model="writeForm.path" size="small" placeholder="/tmp/pass.txt" /></div>
      </div>
      <template #footer>
        <el-button size="small" @click="writeOpen = false">取消</el-button>
        <el-button size="small" type="primary" :loading="busy" @click="doWrite">写入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ListMCPInstalledServices,
  ListMCPServerAliases,
  DeleteMCPInstalledService,
  SaveMCPInstalledService,
  UpdateMCPInstalledNotes,
  WriteMCPFromVault,
} from '../../wailsjs/go/app/App'

export default {
  name: 'ServiceCredentialsPanel',
  props: {
    highlightId: { type: String, default: '' },
  },
  emits: ['meta', 'jump-sensitive'],
  setup(props, { emit, expose }) {
    const loading = ref(false)
    const busy = ref(false)
    const rows = ref([])
    const serverAliases = ref([])
    const serverFilter = ref('')
    const addOpen = ref(false)
    const notesOpen = ref(false)
    const writeOpen = ref(false)
    const addForm = reactive({
      server: '',
      label: '',
      kind: 'credential',
      field: 'password',
      value: '',
      notes: '',
    })
    const notesForm = reactive({
      id: '',
      label: '',
      notes: '',
    })
    const writeForm = reactive({
      vaultId: '',
      server: '',
      field: '',
      fields: [],
      path: '/tmp/pass.txt',
    })

    const emitMeta = () => {
      emit('meta', { count: rows.value.length, loading: loading.value })
    }

    const rowClass = ({ row }) => (props.highlightId && row.id === props.highlightId ? 'sc-hl' : '')

    const loadAliases = async () => {
      serverAliases.value = (await ListMCPServerAliases().catch(() => [])) || []
    }

    const reload = async () => {
      loading.value = true
      emitMeta()
      try {
        const filter = serverFilter.value || ''
        if (filter === '__shared__') {
          const all = (await ListMCPInstalledServices('').catch(() => [])) || []
          rows.value = all.filter((r) => !r.serverAlias)
        } else {
          rows.value = (await ListMCPInstalledServices(filter).catch(() => [])) || []
        }
      } finally {
        loading.value = false
        emitMeta()
      }
    }

    const openAdd = async () => {
      await loadAliases()
      addForm.server = serverFilter.value && serverFilter.value !== '__shared__' ? serverFilter.value : ''
      addForm.label = ''
      addForm.kind = 'credential'
      addForm.field = 'password'
      addForm.value = ''
      addForm.notes = ''
      addOpen.value = true
    }

    const doAdd = async () => {
      if (!String(addForm.value || '').trim()) {
        ElMessage.warning('敏感值不能为空')
        return
      }
      busy.value = true
      try {
        const saved = await SaveMCPInstalledService({
          server: String(addForm.server || '').trim(),
          kind: addForm.kind || 'credential',
          label: addForm.label.trim() || addForm.kind || 'credential',
          field: addForm.field.trim() || 'password',
          value: addForm.value,
          notes: addForm.notes || '',
        })
        ElMessage.success(`已添加 ${saved?.id || ''}${saved?.serverAlias ? '' : '（共用）'}`)
        addOpen.value = false
        addForm.value = ''
        await reload()
      } catch (e) {
        ElMessage.error(`添加失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const openNotes = (row) => {
      notesForm.id = row.id
      notesForm.label = row.label || ''
      notesForm.notes = row.notes || ''
      notesOpen.value = true
    }

    const doNotes = async () => {
      busy.value = true
      try {
        await UpdateMCPInstalledNotes(notesForm.id, notesForm.notes || '')
        ElMessage.success('备注已保存')
        notesOpen.value = false
        await reload()
      } catch (e) {
        ElMessage.error(`保存备注失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const openWrite = async (row) => {
      await loadAliases()
      writeForm.vaultId = row.id
      writeForm.server = row.serverAlias || serverAliases.value[0] || ''
      writeForm.fields = [...(row.secretFields || [])]
      writeForm.field = writeForm.fields[0] || ''
      writeForm.path = '/tmp/pass.txt'
      writeOpen.value = true
    }

    const doWrite = async () => {
      if (!writeForm.server || !writeForm.path || !writeForm.vaultId) {
        ElMessage.warning('目标服务器、路径、凭据不能为空')
        return
      }
      busy.value = true
      try {
        const r = await WriteMCPFromVault(writeForm.server, writeForm.path, writeForm.vaultId, writeForm.field || '')
        ElMessage.success(`已写入 ${r?.path || writeForm.path}（${r?.bytes || 0} 字节）`)
        writeOpen.value = false
      } catch (e) {
        ElMessage.error(`写入失败: ${e}`)
      } finally {
        busy.value = false
      }
    }

    const onDelete = async (row) => {
      try {
        await ElMessageBox.confirm(
          `删除服务凭据「${row.label || row.id}」？仅清本地台账，不动远端。`,
          '确认删除',
          { type: 'warning' },
        )
        await DeleteMCPInstalledService(row.id)
        ElMessage.success('已删除')
        await reload()
      } catch (e) {
        if (e !== 'cancel') ElMessage.error(`删除失败: ${e}`)
      }
    }

    onMounted(() => { loadAliases() })

    reload()
    expose({ reload, openAdd })

    return {
      loading, busy, rows, serverAliases, serverFilter,
      addOpen, notesOpen, writeOpen, addForm, notesForm, writeForm,
      rowClass, reload, openAdd, doAdd, openNotes, doNotes, openWrite, doWrite, onDelete,
    }
  },
}
</script>

<style scoped>
.sc-panel { display: flex; flex-direction: column; gap: 8px; min-height: 0; flex: 1 1 auto; }
.sc-tip {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.sc-tip code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}
.sc-filters { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.sc-ops {
  display: inline-flex;
  align-items: center;
  flex-wrap: nowrap;
  white-space: nowrap;
}
.sc-ops :deep(.el-button) {
  margin: 0;
  padding-left: 6px;
  padding-right: 6px;
}
.sc-form { display: flex; flex-direction: column; gap: 10px; }
.sc-form .row {
  display: grid;
  grid-template-columns: 88px 1fr;
  align-items: start;
  gap: 10px;
  font-size: 12px;
}
.sc-form .row > span {
  line-height: 32px;
}
:deep(.sc-hl) > td {
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent) !important;
}
</style>
