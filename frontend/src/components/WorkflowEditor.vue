<template>
  <div class="wf-layout" @keydown.ctrl.s.prevent="saveWorkflow">
    <!-- 左侧：仅脚本列表 -->
    <div class="wf-sidebar">
      <div class="wf-sidebar-title"><span>脚本</span><strong>{{ filteredScripts.length }}</strong></div>
      <label class="wf-search"><UiIcon name="search" :size="14" /><input v-model.trim="scriptQuery" aria-label="搜索可用脚本" placeholder="搜索脚本" /></label>
      <div class="wf-script-list">
        <div
          v-for="s in filteredScripts"
          :key="s.id"
          class="wf-script-item"
          draggable="true"
          @dragstart="onDragStart($event, s)"
        >{{ s.name }}</div>
      </div>
    </div>

    <!-- 画布区域 -->
    <div class="wf-main">
      <!-- 顶部标题栏 + 按钮（仿 ScriptConfig） -->
      <div class="wf-header">
        <input v-model="wfName" placeholder="工作流名称" class="wf-name-input" />
        <div class="wf-actions">
          <span v-if="running" class="ui-status running">运行中</span>
          <button v-if="!running" class="ui-btn primary" :disabled="saving || store.isDirty || !!actionPending" :title="store.isDirty ? '请先保存工作流' : '运行工作流'" @click="runWorkflow"><UiIcon name="play" />{{ actionPending === 'run' ? '启动中' : '运行' }}</button>
          <button v-else class="ui-btn danger" :disabled="!!actionPending" @click="stopWorkflow"><UiIcon name="stop" />{{ actionPending === 'stop' ? '停止中' : '停止' }}</button>
          <button class="ui-btn" :disabled="saving || running" @click="saveWorkflow"><UiIcon name="save" />{{ saving ? '保存中' : '保存' }}</button>
          <button class="ui-icon-btn" :disabled="running" title="自动布局" aria-label="自动布局" @click="autoLayout"><UiIcon name="layout" /></button>
          <button v-if="selectedWfId" class="ui-icon-btn" :disabled="running || !!actionPending" title="复制工作流" aria-label="复制工作流" @click="copyWorkflow"><UiIcon name="copy" /></button>
          <button class="ui-icon-btn" :disabled="!selectedWfId || running" :title="selectedWfId ? '设置定时任务' : '请先保存工作流'" aria-label="设置定时任务" @click="showTimer = true"><UiIcon name="clock" /></button>
          <button v-if="selectedWfId" class="ui-icon-btn danger-icon" :disabled="running || !!actionPending" title="删除工作流" aria-label="删除工作流" @click="showDeleteConfirm = true"><UiIcon name="delete" /></button>
        </div>
      </div>
      <div v-if="actionError" class="workflow-error">{{ actionError }}</div>

      <div class="wf-canvas" @drop="onDrop" @dragover.prevent>
        <VueFlow
          v-model:nodes="nodes"
          v-model:edges="edges"
          :default-edge-options="{ type: 'smoothstep' }"
          fit-view-on-init
          @connect="onConnect"
        >
          <template #node-script="{ data, id }">
            <Handle type="target" :position="Position.Left" />
            <div :class="['wf-node', data.status]" @dblclick="store.setScriptFromWorkflow(data.scriptId)">
              <div class="wf-node-name">{{ data.label }}</div>
              <div class="wf-node-status">{{ statusLabel(data.status) }}</div>
              <button class="wf-node-rm" title="移除节点" aria-label="移除节点" @click.stop="removeNode(id)"><UiIcon name="x" :size="13" /></button>
            </div>
            <Handle type="source" :position="Position.Right" />
          </template>
          <Background />
          <Controls />
        </VueFlow>
      </div>
    </div>
  </div>
  <div v-if="toast" class="ui-toast">{{ toast }}</div>
  <TimerModal v-if="showTimer" :scriptId="-selectedWfId" :show-existing="true" @close="showTimer = false" />
  <ConfirmDialog v-if="showDeleteConfirm" title="删除工作流？" :message="`工作流“${wfName}”及其编排关系将被删除，此操作无法撤销。`" @confirm="deleteWorkflow" @cancel="showDeleteConfirm = false" />
</template>

<script setup>
import { computed, nextTick, ref, onMounted, onUnmounted, watch } from 'vue'
import { VueFlow, useVueFlow, Handle, Position } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'
import { GetScripts, GetWorkflows, SaveWorkflow, DeleteWorkflow, RunWorkflow, StopWorkflow, CopyWorkflow } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import TimerModal from './TimerModal.vue'
import { statusLabel } from '../utils/status.js'
import ConfirmDialog from './ConfirmDialog.vue'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const vueFlow = useVueFlow()

const allScripts = ref([])
const workflows = ref([])
const selectedWfId = ref('')
const wfName = ref('新工作流')
const nodes = ref([])
const edges = ref([])
const running = ref(false)
const showTimer = ref(false)
const showDeleteConfirm = ref(false)
const scriptQuery = ref('')
const saving = ref(false)
const actionPending = ref('')
const actionError = ref('')
let hydrating = true
const filteredScripts = computed(() => {
  const keyword = scriptQuery.value.toLocaleLowerCase()
  return allScripts.value.filter(script => script.name.toLocaleLowerCase().includes(keyword))
})

watch(() => JSON.stringify({
  name: wfName.value,
  nodes: nodes.value.map(node => ({ id: node.id, scriptId: node.data.scriptId, x: node.position.x, y: node.position.y })),
  edges: edges.value.map(edge => ({ source: edge.source, target: edge.target })),
}), () => {
  if (!hydrating) store.markDirty()
})
let nodeCounter = Date.now()

onMounted(async () => {
  hydrating = true
  allScripts.value = await GetScripts() || []
  workflows.value = await GetWorkflows() || []
  if (store.selectedWorkflowId) {
    selectedWfId.value = store.selectedWorkflowId
    await loadWorkflow()
  }
  await finishHydration()
  EventsOn('workflow:node-status', onNodeStatus)
  EventsOn('workflow:status', onWfStatus)
})

watch(() => store.selectedWorkflowId, async (id) => {
  if (id) {
    hydrating = true
    store.clearDirty()
    workflows.value = await GetWorkflows() || []
    selectedWfId.value = id
    await loadWorkflow()
    await finishHydration()
  }
})

watch(() => store.newWorkflowTick, async () => {
  hydrating = true
  store.clearDirty()
  workflows.value = await GetWorkflows() || []
  selectedWfId.value = ''
  wfName.value = '新工作流'
  nodes.value = []
  edges.value = []
  running.value = false
  await finishHydration()
})

async function finishHydration() {
  await nextTick()
  hydrating = false
}

onUnmounted(() => {
  EventsOff('workflow:node-status')
  EventsOff('workflow:status')
})

function onDragStart(e, script) {
  e.dataTransfer.setData('scriptId', script.id)
  e.dataTransfer.setData('scriptName', script.name)
}

function onDrop(e) {
  const scriptId = parseInt(e.dataTransfer.getData('scriptId'))
  const scriptName = e.dataTransfer.getData('scriptName')
  const rect = e.currentTarget.getBoundingClientRect()
  const id = `n${++nodeCounter}`
  nodes.value.push({
    id,
    type: 'script',
    position: { x: e.clientX - rect.left - 60, y: e.clientY - rect.top - 30 },
    data: { label: scriptName, scriptId, status: 'idle' },
  })
}

function onConnect(params) {
  vueFlow.addEdges([{ ...params, type: 'smoothstep' }])
}

function removeNode(id) {
  nodes.value = nodes.value.filter(n => n.id !== id)
  edges.value = edges.value.filter(e => e.source !== id && e.target !== id)
}

function buildGraph() {
  const ns = vueFlow.getNodes.value
  const es = vueFlow.getEdges.value
  return JSON.stringify({
    nodes: ns.map(n => ({ id: n.id, scriptId: n.data.scriptId, x: n.position.x, y: n.position.y })),
    edges: es.map(e => ({ source: e.source, target: e.target })),
  })
}

async function saveWorkflow() {
  if (saving.value) return
  actionError.value = ''
  if (!wfName.value.trim()) { actionError.value = '请输入工作流名称'; return }
  if (!nodes.value.length) { actionError.value = '请至少添加一个脚本节点'; return }
  saving.value = true
  try {
    const id = await SaveWorkflow({ id: selectedWfId.value || 0, name: wfName.value, graph: buildGraph() })
    workflows.value = await GetWorkflows() || []
    selectedWfId.value = id
    store.selectedWorkflowId = id
    store.clearDirty()
    store.refreshScriptList()
    showToast('保存成功')
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    saving.value = false
  }
}

async function copyWorkflow() {
  if (actionPending.value) return
  actionPending.value = 'copy'
  actionError.value = ''
  try {
    const id = await CopyWorkflow(selectedWfId.value)
    workflows.value = await GetWorkflows() || []
    selectedWfId.value = id
    store.selectedWorkflowId = id
    store.clearDirty()
    store.refreshScriptList()
    showToast('复制成功')
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

const toast = ref('')
function showToast(msg) { toast.value = msg; setTimeout(() => { toast.value = '' }, 2000) }

async function loadWorkflow() {
  if (!selectedWfId.value) {
    nodes.value = []; edges.value = []; wfName.value = '新工作流'; return
  }
  const wf = workflows.value.find(w => w.id === selectedWfId.value)
  if (!wf) return
  running.value = store.runningScripts.has(-Number(selectedWfId.value))
  wfName.value = wf.name
  const g = JSON.parse(wf.graph || '{}')
  nodes.value = (g.nodes || []).map(n => {
    const s = allScripts.value.find(x => x.id === n.scriptId)
    return { id: n.id, type: 'script', position: { x: n.x, y: n.y }, data: { label: s?.name || n.scriptId, scriptId: n.scriptId, status: 'idle' } }
  })
  edges.value = (g.edges || []).map((e, i) => ({ id: `e${i}`, source: e.source, target: e.target, type: 'smoothstep' }))
}

async function runWorkflow() {
  if (!selectedWfId.value) { showToast('请先保存工作流'); return }
  if (store.isDirty || actionPending.value) return
  actionPending.value = 'run'
  actionError.value = ''
  running.value = true
  nodes.value.forEach(n => { n.data = { ...n.data, status: 'idle' } })
  try {
    await RunWorkflow(selectedWfId.value)
  } catch (error) {
    running.value = false
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function stopWorkflow() {
  if (actionPending.value) return
  actionPending.value = 'stop'
  actionError.value = ''
  try {
    await StopWorkflow(selectedWfId.value)
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function deleteWorkflow() {
  if (actionPending.value) return
  actionPending.value = 'delete'
  showDeleteConfirm.value = false
  actionError.value = ''
  try {
    await DeleteWorkflow(selectedWfId.value)
    store.clearDirty()
    selectedWfId.value = ''
    workflows.value = await GetWorkflows() || []
    nodes.value = []; edges.value = []
    store.refreshScriptList()
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

function autoLayout() {
  // Kahn topological sort → assign layers, then position
  const inDeg = {}
  const succ = {}
  nodes.value.forEach(n => { inDeg[n.id] = 0; succ[n.id] = [] })
  edges.value.forEach(e => { inDeg[e.target]++; succ[e.source].push(e.target) })

  const layers = []
  let queue = nodes.value.filter(n => inDeg[n.id] === 0).map(n => n.id)
  while (queue.length) {
    layers.push([...queue])
    const next = []
    queue.forEach(id => succ[id].forEach(t => { if (--inDeg[t] === 0) next.push(t) }))
    queue = next
  }

  const xGap = 200, yGap = 100
  layers.forEach((layer, li) => {
    layer.forEach((id, ri) => {
      const n = nodes.value.find(x => x.id === id)
      if (n) n.position = { x: li * xGap + 40, y: ri * yGap + 40 }
    })
  })
}

function onNodeStatus({ workflowId, nodeId, status }) {
  if (workflowId !== Number(selectedWfId.value)) return
  const n = nodes.value.find(x => x.id === nodeId)
  if (n) n.data = { ...n.data, status }
}

function onWfStatus({ workflowId }) {
  if (workflowId !== Number(selectedWfId.value)) return
  running.value = false
}

function normalizeError(error) {
  return String(error?.message || error || '操作失败').replace(/^Error:\s*/i, '')
}
</script>

<style>
@import '@vue-flow/core/dist/style.css';
@import '@vue-flow/core/dist/theme-default.css';
@import '@vue-flow/controls/dist/style.css';
</style>

<style scoped>
.wf-layout { display: flex; height: 100%; }
.wf-sidebar { width: 160px; min-width: 160px; background: var(--sidebar-bg); border-right: 1px solid var(--border); display: flex; flex-direction: column; padding: 12px; gap: 8px; }
.wf-sidebar-title { font-size: var(--type-label); color: var(--text-muted); }
.wf-script-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 4px; }
.wf-script-item { padding: 7px 10px; background: var(--surface2); border: 1px solid var(--border); border-radius: 4px; font-size: var(--type-label); color: var(--text); cursor: grab; user-select: none; }
.wf-script-item:hover { background: var(--surface); border-color: var(--accent); }
.wf-main { position: relative; flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.wf-header { display: flex; align-items: center; gap: 10px; padding: 10px 14px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border); flex-shrink: 0; }
.wf-name-input { flex: 1; padding: 5px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: var(--type-section-title); font-weight: var(--weight-semibold); max-width: 240px; }
.wf-actions { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.wf-canvas { flex: 1; overflow: hidden; }
.wf-node { position: relative; padding: 12px 16px; background: var(--surface); border: 2px solid var(--accent); border-radius: 8px; min-width: 120px; text-align: center; cursor: default; }
.wf-node-name { font-size: var(--type-body); color: var(--text); font-weight: var(--weight-medium); }
.wf-node-status { font-size: var(--type-caption); color: var(--text-muted); margin-top: 4px; min-height: 14px; }
.wf-node-rm { position: absolute; top: 2px; right: 4px; background: none; border: none; color: var(--text-muted); font-size: var(--type-label); cursor: pointer; padding: 0; }

.wf-sidebar { width: 190px; min-width: 190px; padding: 10px; gap: 8px; }
.wf-sidebar-title { display: flex; align-items: center; justify-content: space-between; color: var(--text-dim); font-size: var(--type-label); font-weight: var(--weight-medium); }
.wf-sidebar-title strong { color: var(--text-muted); font-size: var(--type-caption); font-weight: var(--weight-medium); }
.wf-search { height: 30px; display: flex; align-items: center; gap: 6px; padding: 0 8px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--text-muted); }
.wf-search:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.wf-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--text); font-size: var(--type-label); }
.wf-script-list { gap: 3px; }
.wf-script-item { padding: 7px 9px; border-color: transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); }
.wf-script-item:hover { border-color: var(--border); background: var(--surface-hover); color: var(--text); }
.wf-header { min-height: 54px; gap: 12px; padding: 9px 14px; background: var(--sidebar-bg); }
.wf-name-input { height: 34px; max-width: 280px; padding: 0 10px; border-radius: var(--radius); font-size: var(--type-object-title); }
.wf-name-input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.wf-actions { gap: 5px; }
.danger-icon:hover { border-color: rgba(248, 81, 73, .35); background: var(--red-dim); color: var(--red); }
.workflow-error { position: absolute; z-index: 5; top: 62px; right: 14px; max-width: 420px; padding: 8px 11px; border: 1px solid rgba(248, 81, 73, .28); border-radius: var(--radius); background: var(--surface-raised); color: var(--red); box-shadow: 0 8px 24px rgba(0, 0, 0, .18); font-size: var(--type-label); }
.wf-node { min-width: 136px; padding: 13px 18px; border: 1px solid var(--border-strong); border-left: 3px solid var(--accent); border-radius: var(--radius); background: var(--surface-raised); box-shadow: 0 4px 14px rgba(0, 0, 0, .12); }
.wf-node.running { border-color: var(--orange); background: var(--orange-dim); }
.wf-node.success { border-color: var(--green); background: var(--green-dim); }
.wf-node.error, .wf-node.timeout { border-color: var(--red); background: var(--red-dim); }
.wf-node-rm { width: 22px; height: 22px; display: grid; place-items: center; top: 2px; right: 2px; border-radius: var(--radius-sm); opacity: 0; }
.wf-node:hover .wf-node-rm, .wf-node-rm:focus-visible { opacity: 1; }
.wf-node-rm:hover { background: var(--red-dim); color: var(--red); }
@media (max-width: 920px) {
  .wf-sidebar { width: 160px; min-width: 160px; }
  .wf-actions .ui-btn { padding: 0 10px; }
}
</style>
