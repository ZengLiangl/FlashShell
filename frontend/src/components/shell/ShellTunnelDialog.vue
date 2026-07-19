<template>
  <el-dialog v-model="visible" title="端口转发" width="560px" append-to-body @open="refresh">
    <div class="tunnel-actions">
      <el-button size="small" type="primary" @click="showAdd = true">添加临时隧道</el-button>
      <el-button size="small" :loading="loading" @click="refresh">刷新</el-button>
    </div>
    <el-table :data="tunnels" size="small" empty-text="暂无隧道" max-height="320">
      <el-table-column label="名称" prop="name" min-width="80" />
      <el-table-column label="类型" width="72">
        <template #default="{ row }">{{ typeLabel(row.type) }}</template>
      </el-table-column>
      <el-table-column label="映射" min-width="180">
        <template #default="{ row }">{{ formatRoute(row) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="72">
        <template #default="{ row }">
          <el-tag :type="row.active ? 'success' : 'danger'" size="small">
            {{ row.active ? '运行' : '停止' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="72" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.temporary"
            size="small"
            text
            type="danger"
            @click="removeTunnel(row.name)"
          >移除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showAdd" title="添加临时隧道" width="420px" append-to-body>
      <el-form label-width="88px" size="small">
        <el-form-item label="类型">
          <el-select v-model="form.type">
            <el-option label="本地转发" value="local" />
            <el-option label="远程转发" value="remote" />
            <el-option label="动态 SOCKS" value="dynamic" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="可选" />
        </el-form-item>
        <el-form-item label="本地地址">
          <el-input v-model="form.localHost" placeholder="127.0.0.1" />
        </el-form-item>
        <el-form-item label="本地端口">
          <el-input-number v-model="form.localPort" :min="1" :max="65535" />
        </el-form-item>
        <template v-if="form.type !== 'dynamic'">
          <el-form-item label="远端地址">
            <el-input v-model="form.remoteHost" placeholder="127.0.0.1" />
          </el-form-item>
          <el-form-item label="远端端口">
            <el-input-number v-model="form.remotePort" :min="1" :max="65535" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="showAdd = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="addTunnel">添加</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'ShellTunnelDialog',
  props: {
    modelValue: { type: Boolean, default: false },
    sessionId: { type: String, default: '' },
    configName: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'changed'],
  setup(props, { emit }) {
    const visible = ref(false)
    const loading = ref(false)
    const adding = ref(false)
    const showAdd = ref(false)
    const tunnels = ref([])
    const form = ref({
      type: 'local',
      name: '',
      localHost: '127.0.0.1',
      localPort: 8080,
      remoteHost: '127.0.0.1',
      remotePort: 80,
    })

    watch(() => props.modelValue, (v) => { visible.value = v }, { immediate: true })
    watch(visible, (v) => emit('update:modelValue', v))

    const typeLabel = (t) => ({ local: '本地', remote: '远程', dynamic: 'SOCKS' }[t] || t)
    const formatRoute = (t) => {
      if (t.type === 'dynamic') return `${t.localHost || '127.0.0.1'}:${t.localPort} (SOCKS5)`
      const local = `${t.localHost || '127.0.0.1'}:${t.localPort}`
      const remote = `${t.remoteHost || '127.0.0.1'}:${t.remotePort}`
      return t.type === 'remote' ? `${remote} → ${local}` : `${local} → ${remote}`
    }

    const refresh = async () => {
      const name = props.configName || props.sessionId
      if (!name) return
      loading.value = true
      try {
        tunnels.value = (await App.GetShellTunnelStatus(name)) || []
      } catch {
        tunnels.value = []
      } finally {
        loading.value = false
      }
    }

    const addTunnel = async () => {
      if (!props.sessionId) return
      adding.value = true
      try {
        await App.AddShellTemporaryTunnel(props.sessionId, {
          enabled: true,
          type: form.value.type,
          name: form.value.name || `temp-${Date.now()}`,
          localHost: form.value.localHost || '127.0.0.1',
          localPort: form.value.localPort,
          remoteHost: form.value.remoteHost || '127.0.0.1',
          remotePort: form.value.remotePort,
        })
        showAdd.value = false
        await refresh()
        emit('changed')
        ElMessage.success('隧道已添加')
      } catch (e) {
        ElMessage.error('添加失败: ' + e)
      } finally {
        adding.value = false
      }
    }

    const removeTunnel = async (name) => {
      try {
        await App.RemoveShellTunnel(props.sessionId, name)
        await refresh()
        emit('changed')
      } catch (e) {
        ElMessage.error('移除失败: ' + e)
      }
    }

    return {
      visible, loading, adding, showAdd, tunnels, form,
      typeLabel, formatRoute, refresh, addTunnel, removeTunnel,
    }
  },
}
</script>

<style scoped>
.tunnel-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
</style>
