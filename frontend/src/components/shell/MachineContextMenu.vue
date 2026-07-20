<template>
  <ul
    v-if="ctx.visible"
    class="ctx-menu"
    :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
    @mousedown.stop
    @click.stop
    @mouseleave="$emit('hide')"
  >
    <li @click="onCopy">复制</li>
    <li @click="onEdit">编辑</li>
    <li class="danger" @click="onDelete">删除</li>
  </ul>
</template>

<script>
export default {
  name: 'MachineContextMenu',
  props: {
    ctx: { type: Object, required: true },
  },
  emits: ['copy', 'edit', 'delete', 'hide'],
  setup(props, { emit }) {
    const machine = () => props.ctx.machine

    const onCopy = () => emit('copy', machine())
    const onEdit = () => emit('edit', machine())
    const onDelete = () => emit('delete', machine())

    return { onCopy, onEdit, onDelete }
  },
}
</script>
