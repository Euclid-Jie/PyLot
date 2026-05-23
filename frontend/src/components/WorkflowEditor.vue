<template>
  <div class="wf-layout">
    <!-- 左侧：仅脚本列表 -->
    <div class="wf-sidebar">
      <div class="wf-sidebar-title">脚本</div>
      <div class="wf-script-list">
        <div
          v-for="s in allScripts"
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
          <span v-if="running" class="badge-running">● 运行中</span>
          <button v-if="!running" class="btn-run" @click="runWorkflow">▶ 运行</button>
          <button class="btn-save" @click="saveWorkflow">保存</button>
          <button class="btn-layout" @click="autoLayout">整理布局</button>
          <button v-if="selectedWfId" class="btn-copy" @click="copyWorkflow">复制</button>
          <button class="btn-timer" @click="showTimer = true">⏰ 定时</button>
          <button v-if="selectedWfId" class="btn-delete" @click="deleteWorkflow">删除</button>
        </div>
      </div>

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
              <button class="wf-node-rm" @click.stop="removeNode(id)">✕</button>
            </div>
            <Handle type="source" :position="Position.Right" />
          </template>
          <Background />
          <Controls />
        </VueFlow>
      </div>
    </div>
  </div>
  <div v-if="toast" class="toast">{{ toast }}</div>
  <TimerModal v-if="showTimer" :scriptId="-selectedWfId" @close="showTimer = false" />
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { VueFlow, useVueFlow, Handle, Position } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'
import { GetScripts, GetWorkflows, SaveWorkflow, DeleteWorkflow, RunWorkflow, CopyWorkflow } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import TimerModal from './TimerModal.vue'
import { statusLabel } from '../utils/status.js'

const store = useMainStore()
const { addEdges } = useVueFlow()

const allScripts = ref([])
const workflows = ref([])
const selectedWfId = ref('')
const wfName = ref('新工作流')
const nodes = ref([])
const edges = ref([])
const running = ref(false)
const showTimer = ref(false)
let nodeCounter = Date.now()

onMounted(async () => {
  allScripts.value = await GetScripts() || []
  workflows.value = await GetWorkflows() || []
  if (store.selectedWorkflowId) {
    selectedWfId.value = store.selectedWorkflowId
    await loadWorkflow()
  }
  EventsOn('workflow:node-status', onNodeStatus)
  EventsOn('workflow:status', onWfStatus)
})

watch(() => store.selectedWorkflowId, async (id) => {
  if (id) {
    workflows.value = await GetWorkflows() || []
    selectedWfId.value = id
    await loadWorkflow()
  }
})

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
  addEdges([{ ...params, type: 'smoothstep' }])
}

function removeNode(id) {
  nodes.value = nodes.value.filter(n => n.id !== id)
  edges.value = edges.value.filter(e => e.source !== id && e.target !== id)
}

function buildGraph() {
  return JSON.stringify({
    nodes: nodes.value.map(n => ({ id: n.id, scriptId: n.data.scriptId, x: n.position.x, y: n.position.y })),
    edges: edges.value.map(e => ({ source: e.source, target: e.target })),
  })
}

async function saveWorkflow() {
  const id = await SaveWorkflow({ id: selectedWfId.value || 0, name: wfName.value, graph: buildGraph() })
  workflows.value = await GetWorkflows() || []
  selectedWfId.value = id
  store.selectedWorkflowId = id
  store.refreshScriptList()
  showToast('保存成功')
}

async function copyWorkflow() {
  const id = await CopyWorkflow(selectedWfId.value)
  workflows.value = await GetWorkflows() || []
  selectedWfId.value = id
  store.selectedWorkflowId = id
  store.refreshScriptList()
  showToast('复制成功')
}

const toast = ref('')
function showToast(msg) { toast.value = msg; setTimeout(() => { toast.value = '' }, 2000) }

async function loadWorkflow() {
  if (!selectedWfId.value) {
    nodes.value = []; edges.value = []; wfName.value = '新工作流'; return
  }
  const wf = workflows.value.find(w => w.id === selectedWfId.value)
  if (!wf) return
  wfName.value = wf.name
  const g = JSON.parse(wf.graph || '{}')
  nodes.value = (g.nodes || []).map(n => {
    const s = allScripts.value.find(x => x.id === n.scriptId)
    return { id: n.id, type: 'script', position: { x: n.x, y: n.y }, data: { label: s?.name || n.scriptId, scriptId: n.scriptId, status: 'idle' } }
  })
  edges.value = (g.edges || []).map((e, i) => ({ id: `e${i}`, source: e.source, target: e.target, type: 'smoothstep' }))
}

async function runWorkflow() {
  if (!selectedWfId.value) { alert('请先保存工作流'); return }
  running.value = true
  nodes.value.forEach(n => { n.data = { ...n.data, status: 'idle' } })
  await RunWorkflow(selectedWfId.value)
}

async function deleteWorkflow() {
  if (!confirm('确认删除此工作流？')) return
  await DeleteWorkflow(selectedWfId.value)
  selectedWfId.value = ''
  workflows.value = await GetWorkflows() || []
  nodes.value = []; edges.value = []
  store.refreshScriptList()
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

function onNodeStatus({ nodeId, status }) {
  const n = nodes.value.find(x => x.id === nodeId)
  if (n) n.data = { ...n.data, status }
}

function onWfStatus() {
  running.value = false
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
.wf-sidebar-title { font-size: 12px; color: var(--text-muted); }
.wf-script-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 4px; }
.wf-script-item { padding: 7px 10px; background: var(--surface2); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; color: var(--text); cursor: grab; user-select: none; }
.wf-script-item:hover { background: var(--surface); border-color: var(--accent); }
.wf-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.wf-header { display: flex; align-items: center; gap: 10px; padding: 10px 14px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border); flex-shrink: 0; }
.wf-name-input { flex: 1; padding: 5px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: 14px; font-weight: 600; max-width: 240px; }
.wf-actions { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.badge-running { background: #1b5e20; color: #4caf50; padding: 3px 10px; border-radius: 12px; font-size: 12px; }
.btn-run    { background: #2e7d32; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-save   { background: var(--accent); color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-layout { background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-copy   { background: #6a1b9a; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-timer  { background: #e65100; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-delete { background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.wf-canvas { flex: 1; overflow: hidden; }
.wf-node { position: relative; padding: 12px 16px; background: var(--surface); border: 2px solid var(--accent); border-radius: 8px; min-width: 120px; text-align: center; cursor: default; }
.wf-node.running { border-color: #ff9800; background: #2d1f00; }
.wf-node.success { border-color: #2e7d32; background: #0a1f0a; }
.wf-node.error   { border-color: #c62828; background: #1f0a0a; }
.wf-node.timeout { border-color: #f57f17; background: #1f1500; }
.wf-node-name { font-size: 13px; color: var(--text); font-weight: 500; }
.wf-node-status { font-size: 11px; color: var(--text-muted); margin-top: 4px; min-height: 14px; }
.wf-node-rm { position: absolute; top: 2px; right: 4px; background: none; border: none; color: var(--text-muted); font-size: 12px; cursor: pointer; padding: 0; }
.wf-node-rm:hover { color: #e74c3c; }
.toast { position: fixed; bottom: 40px; left: 50%; transform: translateX(-50%); background: #2e7d32; color: #fff; padding: 7px 20px; border-radius: 20px; font-size: 13px; z-index: 200; pointer-events: none; }
</style>
