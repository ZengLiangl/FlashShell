<template>
  <div v-if="visible" class="vault-unlock-overlay">
    <div class="vault-unlock-card">
      <template v-if="mode === 'unlock'">
        <h2>解锁凭据库</h2>
        <p class="desc">
          已启用主密码。解锁后才能连接 SSH、查看敏感信息或使用 MCP。
        </p>
        <el-form @submit.prevent="submit">
          <el-input
            v-model="password"
            type="password"
            show-password
            placeholder="主密码"
            size="large"
            autofocus
            @keyup.enter="submit"
          />
          <p v-if="error" class="err">{{ error }}</p>
          <el-button
            type="primary"
            size="large"
            class="unlock-btn"
            :loading="loading"
            @click="submit"
          >
            解锁
          </el-button>
        </el-form>
        <div class="forgot">
          <el-button text type="danger" @click="mode = 'reset'">
            忘记主密码？重置凭据库
          </el-button>
          <p class="hint">
            无恢复机制：将删除全部服务器密码/私钥密文，服务器列表保留；之后需重新填写凭据。
          </p>
        </div>
      </template>

      <template v-else>
        <h2>重置凭据库</h2>
        <p class="desc warn">
          将永久删除全部已保存的服务器密码与私钥密文（服务器列表/审计保留）。
          此操作不需要旧主密码，且无法撤销。
        </p>
        <el-input
          v-model="resetConfirm"
          placeholder="请输入大写 RESET 确认"
          size="large"
          autofocus
          @keyup.enter="doReset"
        />
        <p v-if="error" class="err">{{ error }}</p>
        <div class="reset-actions">
          <el-button size="large" :disabled="resetting" @click="cancelReset">取消</el-button>
          <el-button
            type="danger"
            size="large"
            :loading="resetting"
            :disabled="resetConfirm !== 'RESET'"
            @click="doReset"
          >
            确认重置
          </el-button>
        </div>
      </template>
    </div>
  </div>
</template>

<script>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { GetVaultStatus, UnlockVault, ResetVaultForgotMasterPassword } from '../../wailsjs/go/app/App'

export default {
  name: 'VaultUnlockOverlay',
  setup() {
    const visible = ref(false)
    const mode = ref('unlock')
    const password = ref('')
    const resetConfirm = ref('')
    const error = ref('')
    const loading = ref(false)
    const resetting = ref(false)
    let offStatus = null

    const applyStatus = (st) => {
      const locked = st && st.hasMasterPassword && !st.unlocked
      visible.value = !!locked
      if (!locked) {
        password.value = ''
        error.value = ''
        mode.value = 'unlock'
        resetConfirm.value = ''
      }
    }

    const refresh = async () => {
      try {
        applyStatus(await GetVaultStatus())
      } catch {
        visible.value = false
      }
    }

    const submit = async () => {
      loading.value = true
      error.value = ''
      try {
        await UnlockVault(password.value || '')
        password.value = ''
        await refresh()
      } catch (e) {
        error.value = String(e?.message || e || '解锁失败')
      } finally {
        loading.value = false
      }
    }

    const cancelReset = () => {
      mode.value = 'unlock'
      resetConfirm.value = ''
      error.value = ''
    }

    const doReset = async () => {
      if (resetConfirm.value !== 'RESET') {
        error.value = '请输入大写 RESET'
        return
      }
      resetting.value = true
      error.value = ''
      try {
        await ResetVaultForgotMasterPassword()
        password.value = ''
        resetConfirm.value = ''
        mode.value = 'unlock'
        await refresh()
      } catch (e) {
        error.value = String(e?.message || e || '重置失败')
      } finally {
        resetting.value = false
      }
    }

    onMounted(() => {
      refresh()
      offStatus = EventsOn('vault:status', applyStatus)
    })
    onBeforeUnmount(() => {
      offStatus?.()
    })

    watch(visible, (v) => {
      if (v) {
        password.value = ''
        mode.value = 'unlock'
        resetConfirm.value = ''
        error.value = ''
      }
    })

    return {
      visible,
      mode,
      password,
      resetConfirm,
      error,
      loading,
      resetting,
      submit,
      cancelReset,
      doReset,
    }
  },
}
</script>

<style scoped>
/* 须低于 Element Plus 弹层起始 z-index（约 2000），否则退出确认等 MessageBox 会被挡住 */
.vault-unlock-overlay {
  position: fixed;
  inset: 0;
  z-index: 1800;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, #0b1220 72%, transparent);
  backdrop-filter: blur(8px);
}
.vault-unlock-card {
  width: min(400px, 92vw);
  padding: 28px 28px 22px;
  border-radius: 14px;
  background: var(--app-card-bg, #1a1f2e);
  border: 1px solid var(--app-border, #2a3348);
  color: var(--app-text, #e8ecf4);
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.35);
}
h2 {
  margin: 0 0 8px;
  font-size: 20px;
  font-weight: 650;
}
.desc {
  margin: 0 0 18px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--app-text-muted, #9aa3b5);
}
.desc.warn {
  color: var(--el-color-danger);
}
.unlock-btn {
  width: 100%;
  margin-top: 14px;
}
.err {
  margin: 10px 0 0;
  color: var(--el-color-danger);
  font-size: 12px;
}
.forgot {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid var(--app-border, #2a3348);
  text-align: center;
}
.hint {
  margin: 8px 0 0;
  font-size: 11px;
  color: var(--app-text-muted, #9aa3b5);
  line-height: 1.45;
  text-align: left;
}
.reset-actions {
  display: flex;
  gap: 10px;
  margin-top: 14px;
}
.reset-actions .el-button {
  flex: 1;
}
</style>
