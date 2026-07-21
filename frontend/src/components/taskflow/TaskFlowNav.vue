<template>
  <aside class="task-flow-nav">
    <div class="nav-toolbar">
      <span class="nav-toolbar-title">项目</span>
      <el-tooltip content="新增项目" placement="top">
        <el-button type="primary" size="small" circle @click="$emit('add-project')">
          <el-icon><Plus /></el-icon>
        </el-button>
      </el-tooltip>
    </div>

    <div class="nav-tree">
      <div v-if="!projects.length" class="nav-empty">
        <p class="nav-empty-title">暂无项目</p>
        <p class="nav-empty-desc">点击右上角 + 新建项目</p>
      </div>

      <div v-for="(project, pIndex) in projects" :key="pIndex" class="nav-project">
        <div
          class="nav-row project-row"
          :class="{ active: isProjectActive(pIndex) }"
          @click="$emit('select-project', pIndex)"
        >
          <button
            type="button"
            class="expand-btn"
            :title="expanded[pIndex] ? '收起' : '展开'"
            @click.stop="toggleProject(pIndex)"
          >
            <el-icon :size="12">
              <ArrowDown v-if="expanded[pIndex]" />
              <ArrowRight v-else />
            </el-icon>
          </button>
          <span
            class="nav-label"
            :title="(project.name || '(未命名项目)') + '（双击展开/收起）'"
            @dblclick.stop="toggleProject(pIndex)"
          >
            {{ project.name || '(未命名项目)' }}
          </span>
          <div class="nav-actions icon-actions" @click.stop>
            <el-tooltip content="新增子项目" placement="top">
              <el-button
                type="primary"
                text
                size="small"
                circle
                @click="onAddSub(pIndex)"
              >
                <el-icon><Plus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip content="删除项目" placement="top">
              <el-button
                type="danger"
                text
                size="small"
                circle
                @click="$emit('remove-project', pIndex)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <div v-show="expanded[pIndex]" class="nav-subs">
          <div
            v-for="(sub, sIndex) in project.subprojects || []"
            :key="sIndex"
            class="nav-row sub-row"
            :class="{ active: isSubActive(pIndex, sIndex) }"
            @click="$emit('select-sub', pIndex, sIndex)"
          >
            <span class="nav-label" :title="sub.name || '(未命名子项目)'">
              {{ sub.name || '(未命名子项目)' }}
            </span>
            <el-tag size="small" type="info" effect="plain" class="cmd-count">
              {{ (sub.commands || []).length }}
            </el-tag>
            <div class="nav-actions icon-actions" @click.stop>
              <el-tooltip content="删除子项目" placement="top">
                <el-button
                  type="danger"
                  text
                  size="small"
                  circle
                  @click="$emit('remove-sub', pIndex, sIndex)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </div>
          <div v-if="!(project.subprojects || []).length" class="nav-subs-empty">
            暂无子项目，点项目行 + 添加
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script>
import { reactive, watch } from 'vue'
import { ArrowDown, ArrowRight, Delete, Plus } from '@element-plus/icons-vue'

export default {
  name: 'TaskFlowNav',
  components: { ArrowDown, ArrowRight, Delete, Plus },
  props: {
    projects: { type: Array, default: () => [] },
    selectedP: { type: Number, default: null },
    selectedS: { type: Number, default: null },
  },
  emits: [
    'add-project',
    'remove-project',
    'add-sub',
    'remove-sub',
    'select-project',
    'select-sub',
  ],
  setup(props, { emit }) {
    const expanded = reactive({})

    const ensureExpanded = () => {
      props.projects.forEach((_, i) => {
        if (expanded[i] === undefined) expanded[i] = false
      })
      // 仅当选中子项目时展开，便于看到高亮；仅选中项目不自动展开
      if (props.selectedP != null && props.selectedS != null) {
        expanded[props.selectedP] = true
      }
    }

    watch(
      () => [props.projects.length, props.selectedP, props.selectedS],
      () => ensureExpanded(),
      { immediate: true },
    )

    const toggleProject = (pIndex) => {
      expanded[pIndex] = !expanded[pIndex]
    }

    const onAddSub = (pIndex) => {
      expanded[pIndex] = true
      emit('add-sub', pIndex)
    }

    const isProjectActive = (pIndex) =>
      props.selectedP === pIndex && (props.selectedS == null || props.selectedS < 0)

    const isSubActive = (pIndex, sIndex) =>
      props.selectedP === pIndex && props.selectedS === sIndex

    return {
      expanded,
      toggleProject,
      onAddSub,
      isProjectActive,
      isSubActive,
    }
  },
}
</script>

<style scoped>
.task-flow-nav {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--app-border);
  background: var(--app-bg);
  min-height: 0;
}

.nav-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 12px;
  border-bottom: 1px solid var(--app-border);
  flex-shrink: 0;
  min-height: 44px;
  box-sizing: border-box;
}

.nav-toolbar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  letter-spacing: 0.02em;
}

.nav-tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.nav-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 32px 12px;
  text-align: center;
}

.nav-empty-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text-secondary);
}

.nav-empty-desc {
  margin: 0;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.4;
}

.nav-project {
  margin-bottom: 2px;
}

.nav-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-radius: var(--app-radius-sm, 6px);
  cursor: pointer;
  color: var(--app-text-secondary);
  font-size: 13px;
}

.nav-row:hover {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
}

.nav-row.active {
  background: var(--app-accent-bg);
  color: var(--app-accent-color);
  font-weight: 600;
}

.project-row {
  font-weight: 500;
}

.sub-row {
  padding-left: 28px;
}

.expand-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  flex-shrink: 0;
}

.nav-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-row .nav-label {
  cursor: pointer;
  user-select: none;
}

.cmd-count {
  flex-shrink: 0;
}

.nav-actions {
  flex-shrink: 0;
  opacity: 0;
  gap: 2px;
}

.nav-row:hover .nav-actions,
.nav-row.active .nav-actions,
.nav-row:focus-within .nav-actions {
  opacity: 1;
}

.nav-subs {
  margin-bottom: 4px;
}

.nav-subs-empty {
  margin: 2px 0 8px 28px;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 1.4;
}
</style>
