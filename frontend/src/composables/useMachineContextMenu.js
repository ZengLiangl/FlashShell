import { reactive, onMounted, onUnmounted } from 'vue'

export function useMachineContextMenu() {
  const ctx = reactive({
    visible: false,
    x: 0,
    y: 0,
    machine: null,
  })

  const hideContextMenu = () => {
    ctx.visible = false
    ctx.machine = null
  }

  const onMachineContextMenu = (event, machine) => {
    event.preventDefault()
    event.stopPropagation()
    ctx.x = event.clientX
    ctx.y = event.clientY
    ctx.machine = machine
    ctx.visible = true
  }

  const isContextTarget = (machine) =>
    ctx.visible && ctx.machine && (ctx.machine.id || ctx.machine.name) === (machine.id || machine.name)

  onMounted(() => {
    document.addEventListener('click', hideContextMenu)
  })

  onUnmounted(() => {
    document.removeEventListener('click', hideContextMenu)
  })

  return {
    ctx,
    hideContextMenu,
    onMachineContextMenu,
    isContextTarget,
  }
}
