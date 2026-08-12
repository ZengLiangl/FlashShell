<template>
  <Teleport to="body">
    <ul
      v-if="ctx.visible"
      ref="menuRef"
      class="ctx-menu"
      :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
      @mousedown.stop
      @click.stop
      @mouseleave="$emit('hide')"
    >
      <li v-if="showConnect" @click="onConnect">连接</li>
      <li v-if="showConnect" @click="onOpenWindow">在新窗口打开</li>
      <li @click="onTogglePin">{{ isPinned ? '取消置顶' : '置顶' }}</li>
      <li @click="onCopy">复制</li>
      <li @click="onEdit">编辑</li>
      <li class="danger" @click="onDelete">删除</li>
    </ul>
  </Teleport>
</template>

<script>
import { ref, watch, nextTick, computed } from 'vue'

export default {
  name: 'MachineContextMenu',
  props: {
    ctx: { type: Object, required: true },
    showConnect: { type: Boolean, default: true },
  },
  emits: ['connect', 'open-window', 'copy', 'edit', 'delete', 'toggle-pin', 'hide'],
  setup(props, { emit }) {
    const menuRef = ref(null)

    const machine = () => props.ctx.machine
    const isPinned = computed(() => !!machine()?.pinned)

    const adjustPosition = async () => {
      await nextTick()
      const el = menuRef.value
      if (!el || !props.ctx.visible) return
      const rect = el.getBoundingClientRect()
      const pad = 8
      const vw = window.innerWidth
      const vh = window.innerHeight
      let x = props.ctx.x
      let y = props.ctx.y
      if (x + rect.width > vw - pad) {
        x = Math.max(pad, vw - rect.width - pad)
      }
      if (y + rect.height > vh - pad) {
        y = Math.max(pad, y - rect.height)
      }
      if (y + rect.height > vh - pad) {
        y = Math.max(pad, vh - rect.height - pad)
      }
      props.ctx.x = x
      props.ctx.y = y
    }

    watch(
      () => props.ctx.visible,
      (visible) => {
        if (visible) adjustPosition()
      },
    )

    const onConnect = () => emit('connect', machine())
    const onOpenWindow = () => emit('open-window', machine())
    const onCopy = () => emit('copy', machine())
    const onEdit = () => emit('edit', machine())
    const onDelete = () => emit('delete', machine())
    const onTogglePin = () => emit('toggle-pin', machine())

    return { menuRef, isPinned, onConnect, onOpenWindow, onCopy, onEdit, onDelete, onTogglePin }
  },
}
</script>
