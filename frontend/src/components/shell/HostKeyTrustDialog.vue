<template>
  <el-dialog
    v-model="visible"
    title="信任主机密钥"
    width="480px"
    :close-on-click-modal="false"
    append-to-body
    @closed="onClosed"
  >
    <p class="hk-desc">首次连接该主机，请核对指纹后选择是否信任。</p>
    <dl class="hk-info">
      <dt>主机</dt>
      <dd>{{ info.host }}:{{ info.port }}</dd>
      <dt>指纹</dt>
      <dd class="hk-fp">{{ info.fingerprint }}</dd>
    </dl>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button :loading="trusting" @click="onTrustOnce">只信任本次</el-button>
      <el-button type="primary" :loading="trusting" @click="onTrustSave">信任并保存</el-button>
    </template>
  </el-dialog>
</template>

<script>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import * as App from '../../../wailsjs/go/app/App'

export default {
  name: 'HostKeyTrustDialog',
  props: {
    modelValue: { type: Boolean, default: false },
    hostKeyInfo: { type: Object, default: null },
  },
  emits: ['update:modelValue', 'trusted'],
  setup(props, { emit }) {
    const visible = ref(false)
    const trusting = ref(false)
    const info = ref({ host: '', port: 22, fingerprint: '' })

    watch(
      () => props.modelValue,
      (v) => {
        visible.value = v
        if (v && props.hostKeyInfo) info.value = { ...props.hostKeyInfo }
      },
      { immediate: true },
    )
    watch(visible, (v) => emit('update:modelValue', v))

    const finishTrust = (persistent) => {
      visible.value = false
      emit('trusted', { ...info.value, persistent })
    }

    const onTrustOnce = async () => {
      trusting.value = true
      try {
        await App.TrustHostKeyOnce(info.value.host, info.value.port, info.value.fingerprint)
        ElMessage.success('已信任本次连接')
        finishTrust(false)
      } catch (e) {
        ElMessage.error('操作失败: ' + e)
      } finally {
        trusting.value = false
      }
    }

    const onTrustSave = async () => {
      trusting.value = true
      try {
        await App.TrustHostKey(info.value.host, info.value.port, info.value.fingerprint)
        ElMessage.success('已保存主机密钥')
        finishTrust(true)
      } catch (e) {
        ElMessage.error('保存失败: ' + e)
      } finally {
        trusting.value = false
      }
    }

    const onClosed = () => emit('update:modelValue', false)

    return { visible, trusting, info, onTrustOnce, onTrustSave, onClosed }
  },
}
</script>

<style scoped>
.hk-desc {
  margin: 0 0 12px;
  color: var(--app-text-secondary);
  font-size: 13px;
}
.hk-info {
  margin: 0;
  display: grid;
  grid-template-columns: 56px 1fr;
  gap: 8px 12px;
  font-size: 13px;
}
.hk-info dt {
  color: var(--app-text-secondary);
}
.hk-info dd {
  margin: 0;
  word-break: break-all;
}
.hk-fp {
  font-family: var(--app-mono-font, monospace);
  font-size: 12px;
}
</style>
