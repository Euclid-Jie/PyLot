<template>
  <div class="sidebar-inner" @click="openMenuId = null">
    <div class="brand">
      <div class="brand-mark">P</div>
      <div><strong>PyLot</strong><span>脚本调度中心</span></div>
    </div>

    <nav class="primary-nav" aria-label="主要导航">
      <button :class="{ active: store.currentView === 'schedule' }" @click="store.setView('schedule')"><UiIcon name="clock" /><span>定时任务</span></button>
      <button :class="{ active: store.currentView === 'services' }" @click="store.setView('services')"><UiIcon name="server" /><span>服务</span></button>
    </nav>

    <div class="resource-scroll">
      <section class="resource-section">
        <div class="section-heading">
          <span>脚本列表</span>
          <button class="ui-icon-btn compact" title="新增列表" aria-label="新增列表" :disabled="store.isDirty" @click.stop="beginCreate"><UiIcon name="add" /></button>
        </div>

        <button class="list-row system" :class="{ active: isActiveList('all') }" @click="store.setScriptList('all')">
          <UiIcon name="layout" :size="15" /><span>全部脚本</span><strong>{{ allScripts.length }}</strong>
        </button>

        <div v-for="list in scriptLists" :key="list.id" class="managed-row" :class="{ active: isActiveList(list.id), editing: editingId === list.id }">
          <template v-if="editingId === list.id">
            <UiIcon name="layout" :size="15" />
            <input ref="editInput" v-model.trim="draftName" maxlength="40" @keydown.enter="saveRename(list.id)" @keydown.esc="cancelEdit" @click.stop />
            <button class="inline-action" title="保存" aria-label="保存" @click.stop="saveRename(list.id)"><UiIcon name="save" :size="14" /></button>
          </template>
          <template v-else>
            <button class="list-main" @click="store.setScriptList(list.id)"><UiIcon name="layout" :size="15" /><span>{{ list.name }}</span><strong>{{ list.scriptCount }}</strong></button>
            <button class="row-menu-btn" title="列表操作" aria-label="列表操作" :disabled="store.isDirty" @click.stop="toggleMenu(list.id)"><UiIcon name="more" :size="15" /></button>
            <div v-if="openMenuId === list.id" class="row-menu" @click.stop>
              <button @click="beginRename(list)"><UiIcon name="edit" :size="14" />重命名</button>
              <button :disabled="isFirstList(list.id)" @click="moveList(list.id, -1)"><UiIcon name="chevronUp" :size="14" />上移</button>
              <button :disabled="isLastList(list.id)" @click="moveList(list.id, 1)"><UiIcon name="chevronDown" :size="14" />下移</button>
              <button class="danger" @click="askDelete(list)"><UiIcon name="delete" :size="14" />删除列表</button>
            </div>
          </template>
        </div>

        <form v-if="creating" class="new-list-row" @submit.prevent="createList">
          <UiIcon name="layout" :size="15" />
          <input ref="createInput" v-model.trim="draftName" maxlength="40" placeholder="列表名称" @keydown.esc="cancelEdit" />
          <button class="inline-action" type="submit" title="创建" aria-label="创建"><UiIcon name="add" :size="14" /></button>
        </form>

        <button class="list-row system" :class="{ active: isActiveList('unassigned') }" @click="store.setScriptList('unassigned')">
          <UiIcon name="folder" :size="15" /><span>未分类</span><strong>{{ unassignedCount }}</strong>
        </button>
        <p v-if="listError" class="list-error">{{ listError }}</p>
      </section>

      <section class="resource-section">
        <div class="section-title-row">
          <button class="section-title" @click="workflowsOpen = !workflowsOpen"><UiIcon :name="workflowsOpen ? 'chevronDown' : 'chevronRight'" :size="14" /><span>工作流</span><strong>{{ allWorkflows.length }}</strong></button>
          <button class="ui-icon-btn compact" aria-label="新建工作流" title="新建工作流" @click="store.newWorkflow()"><UiIcon name="add" /></button>
        </div>
        <div v-if="workflowsOpen">
          <button v-for="workflow in allWorkflows" :key="workflow.id" class="workflow-item" :class="{ active: store.selectedWorkflowId === workflow.id && store.currentView === 'workflow' }" :title="workflow.name" @click="store.openWorkflow(workflow.id)">
            <UiIcon name="flow" :size="15" /><span>{{ workflow.name }}</span><i v-if="store.runningScripts.has(-workflow.id)" class="running-dot"></i>
          </button>
          <div v-if="!allWorkflows.length" class="resource-empty">暂无工作流</div>
        </div>
      </section>
    </div>

    <footer><button :class="{ active: store.currentView === 'settings' }" @click="store.setView('settings')"><UiIcon name="settings" /><span>设置</span></button></footer>
  </div>

  <ConfirmDialog
    v-if="deleteTarget"
    title="删除列表？"
    :message="deleteTarget.scriptCount ? `列表“${deleteTarget.name}”将被删除，其中 ${deleteTarget.scriptCount} 个脚本会移入“未分类”。` : `列表“${deleteTarget.name}”将被删除。`"
    confirm-text="删除列表"
    @confirm="deleteList"
    @cancel="deleteTarget = null"
  />
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { CreateScriptList, DeleteScriptList, GetScriptLists, GetScripts, GetWorkflows, MoveScriptList, RenameScriptList } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import ConfirmDialog from './ConfirmDialog.vue'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const allScripts = ref([])
const scriptLists = ref([])
const allWorkflows = ref([])
const workflowsOpen = ref(true)
const creating = ref(false)
const editingId = ref(null)
const openMenuId = ref(null)
const deleteTarget = ref(null)
const draftName = ref('')
const listError = ref('')
const createInput = ref(null)
const editInput = ref(null)

const unassignedCount = computed(() => allScripts.value.filter(script => !script.listId).length)

onMounted(loadResources)
watch(() => store.scriptListVersion, loadResources)

async function loadResources() {
  const [scripts, lists, workflows] = await Promise.all([GetScripts(), GetScriptLists(), GetWorkflows()])
  allScripts.value = scripts || []
  scriptLists.value = lists || []
  allWorkflows.value = workflows || []
  if (typeof store.selectedScriptListId === 'number' && !scriptLists.value.some(item => item.id === store.selectedScriptListId)) {
    store.setScriptList('all')
  }
}

function isActiveList(id) { return store.currentView === 'script' && store.selectedScriptListId === id }
function isFirstList(id) { return scriptLists.value[0]?.id === id }
function isLastList(id) { return scriptLists.value[scriptLists.value.length - 1]?.id === id }
function toggleMenu(id) { openMenuId.value = openMenuId.value === id ? null : id }

async function beginCreate() {
  creating.value = true
  editingId.value = null
  draftName.value = ''
  listError.value = ''
  await nextTick()
  createInput.value?.focus()
}

async function beginRename(list) {
  openMenuId.value = null
  creating.value = false
  editingId.value = list.id
  draftName.value = list.name
  listError.value = ''
  await nextTick()
  const input = Array.isArray(editInput.value) ? editInput.value[0] : editInput.value
  input?.focus()
  input?.select()
}

function cancelEdit() {
  creating.value = false
  editingId.value = null
  draftName.value = ''
}

async function createList() {
  if (!draftName.value) return
  listError.value = ''
  try {
    const id = await CreateScriptList(draftName.value)
    cancelEdit()
    store.refreshScriptList()
    store.setScriptList(id)
  } catch (error) {
    listError.value = normalizeError(error)
  }
}

async function saveRename(id) {
  if (!draftName.value) return
  listError.value = ''
  try {
    await RenameScriptList(id, draftName.value)
    cancelEdit()
    store.refreshScriptList()
  } catch (error) {
    listError.value = normalizeError(error)
  }
}

async function moveList(id, direction) {
  openMenuId.value = null
  listError.value = ''
  try {
    await MoveScriptList(id, direction)
    store.refreshScriptList()
  } catch (error) {
    listError.value = normalizeError(error)
  }
}

function askDelete(list) {
  openMenuId.value = null
  deleteTarget.value = list
}

async function deleteList() {
  const target = deleteTarget.value
  deleteTarget.value = null
  listError.value = ''
  try {
    await DeleteScriptList(target.id)
    if (store.selectedScriptListId === target.id) store.setScriptList('unassigned')
    store.refreshScriptList()
  } catch (error) {
    listError.value = normalizeError(error)
  }
}

function normalizeError(error) {
  return String(error?.message || error || '操作失败').replace(/^Error:\s*/i, '')
}
</script>

<style scoped>
.sidebar-inner { height: 100%; display: flex; flex-direction: column; }
.brand { min-height: 66px; display: flex; align-items: center; gap: 11px; padding: 12px 14px; border-bottom: 1px solid var(--border); }
.brand-mark { width: 34px; height: 34px; display: grid; place-items: center; flex-shrink: 0; border-radius: 8px; background: var(--accent); color: #fff; font-size: 17px; font-weight: var(--weight-bold); }
.brand strong, .brand span { display: block; }
.brand strong { color: var(--text); font-size: var(--type-section-title); font-weight: var(--weight-semibold); }
.brand span { margin-top: 1px; color: var(--text-muted); font-size: var(--type-label); }
.primary-nav { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 10px 10px 4px; }
.primary-nav button, footer button { min-height: 36px; display: flex; align-items: center; gap: 8px; padding: 0 11px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); font-size: var(--type-body); text-align: left; }
.primary-nav button:hover, footer button:hover { background: var(--surface-hover); color: var(--text); }
.primary-nav button.active, footer button.active { border-color: rgba(59, 130, 246, .28); background: var(--accent-dim); color: var(--accent); font-weight: var(--weight-medium); }
.resource-scroll { min-height: 0; flex: 1; overflow-y: auto; padding: 8px 8px 12px; }
.resource-section + .resource-section { margin-top: 10px; padding-top: 9px; border-top: 1px solid var(--border); }
.section-heading { min-height: 34px; display: flex; align-items: center; justify-content: space-between; padding: 0 5px 0 7px; color: var(--text-muted); font-size: var(--type-label); font-weight: var(--weight-medium); }
.compact { width: 26px; height: 26px; }
.list-row { width: 100%; min-height: 34px; display: grid; grid-template-columns: 18px minmax(0, 1fr) auto; align-items: center; gap: 7px; padding: 0 9px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); font-size: var(--type-body); text-align: left; }
.list-row span, .list-main span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.list-row strong, .list-main strong { color: var(--text-muted); font-size: var(--type-caption); font-weight: var(--weight-medium); }
.list-row:hover, .managed-row:hover { background: var(--surface-hover); color: var(--text); }
.list-row.active, .managed-row.active { border-color: rgba(59, 130, 246, .22); background: var(--accent-dim); color: var(--accent); }
.managed-row { position: relative; min-height: 34px; display: grid; grid-template-columns: minmax(0, 1fr) 27px; align-items: center; border: 1px solid transparent; border-radius: var(--radius); color: var(--text-dim); }
.list-main { min-width: 0; min-height: 32px; display: grid; grid-template-columns: 18px minmax(0, 1fr) auto; align-items: center; gap: 7px; padding: 0 2px 0 8px; border: 0; background: transparent; color: inherit; font-size: var(--type-body); text-align: left; }
.row-menu-btn { width: 26px; height: 26px; display: grid; place-items: center; border: 0; border-radius: var(--radius); background: transparent; color: var(--text-muted); opacity: 0; }
.managed-row:hover .row-menu-btn, .row-menu-btn:focus, .managed-row.active .row-menu-btn { opacity: 1; }
.row-menu-btn:hover { background: var(--surface-raised); color: var(--text); }
.row-menu { position: absolute; z-index: 30; top: 30px; right: 2px; width: 130px; padding: 4px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface-raised); box-shadow: var(--shadow-dialog); }
.row-menu button { width: 100%; min-height: 30px; display: flex; align-items: center; gap: 8px; padding: 0 8px; border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-dim); font-size: var(--type-label); text-align: left; }
.row-menu button:hover { background: var(--surface-hover); color: var(--text); }
.row-menu button:disabled { opacity: .38; }
.row-menu button.danger { color: var(--red); }
.new-list-row, .managed-row.editing { min-height: 36px; display: grid; grid-template-columns: 18px minmax(0, 1fr) 26px; align-items: center; gap: 6px; padding: 0 6px 0 9px; border: 1px solid var(--accent); border-radius: var(--radius); color: var(--accent); }
.new-list-row input, .managed-row input { min-width: 0; height: 27px; border: 0; outline: 0; background: transparent; color: var(--text); font-size: var(--type-body); }
.inline-action { width: 25px; height: 25px; display: grid; place-items: center; border: 0; border-radius: var(--radius-sm); background: var(--accent-dim); color: var(--accent); }
.list-error { margin: 6px 8px 0; color: var(--red); font-size: var(--type-caption); }
.section-title-row { display: flex; align-items: center; }
.section-title-row .section-title { flex: 1; }
.section-title { width: 100%; min-height: 32px; display: grid; grid-template-columns: 16px minmax(0, 1fr) auto; align-items: center; gap: 5px; padding: 0 6px; border: 0; background: transparent; color: var(--text-dim); font-size: var(--type-label); font-weight: var(--weight-medium); text-align: left; }
.section-title strong { color: var(--text-muted); font-size: var(--type-caption); font-weight: var(--weight-medium); }
.workflow-item { width: 100%; min-height: 32px; display: grid; grid-template-columns: 18px minmax(0, 1fr) 10px; align-items: center; gap: 7px; padding: 0 8px 0 16px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); font-size: var(--type-body); text-align: left; }
.workflow-item span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.workflow-item:hover { background: var(--surface-hover); color: var(--text); }
.workflow-item.active { border-color: rgba(59, 130, 246, .22); background: var(--accent-dim); color: var(--accent); }
.running-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--green); box-shadow: 0 0 0 3px var(--green-dim); animation: pulse 1.5s ease-in-out infinite; }
.resource-empty { padding: 10px 16px; color: var(--text-muted); font-size: var(--type-label); }
footer { padding: 9px 10px; border-top: 1px solid var(--border); }
footer button { width: 100%; }
@keyframes pulse { 50% { opacity: .45; } }
</style>
