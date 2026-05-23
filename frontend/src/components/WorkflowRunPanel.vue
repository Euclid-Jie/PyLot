<template>
  <div class="wf-run-panel">
    <div class="config-header">
      <h2>{{ wf?.name || '工作流' }}</h2>
      <div class="header-actions">
        <span v-if="isRunning" class="badge-running">● 运行中</span>
        <button v-if="!isRunning" class="btn-run" @click="run">▶ 运行</button>
        <button v-if="isRunning" class="btn-stop" @click="stop">■ 停止</button>
        <button class="btn-edit" @click="store.setView('workflow')">编辑</button>
      </div>
    </div>
    <div class="node-list">
      <div v-for="node in nodes" :key="node.id" class="node-row"
           :class="{ active: store.selectedScriptID === node.scriptId }"
           @click="store.setScript(node.scriptId)">
        <span :class="['dot', node.status]"></span>
        <span class="node-name">{{ node.name }}</span>
        <span class="node-status">{{ statusLabel(node.status) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'
import { GetWorkflows, RunWorkflow, GetScripts } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import { statusLabel } from '../utils/status.js'

const store = useMainStore()
const wfId = computed(() => -store.selectedScriptID)
const wf = ref(null)
const nodes = ref([])
const isRunning = computed(() => store.runningScripts.has(store.selectedScriptID))

onMounted(async () => {
  const [wfs, scripts] = await Promise.all([GetWorkflows(), GetScripts()])
  wf.value = wfs?.find(w => w.id === wfId.value)
  if (wf.value) {
    const g = JSON.parse(wf.value.graph || '{}')
    nodes.value = (g.nodes || []).map(n => ({
      id: n.id,
      scriptId: n.scriptId,
      name: scripts?.find(s => s.id === n.scriptId)?.name || n.scriptId,
      status: 'idle',
    }))
  }
  EventsOn('workflow:node-status', ({ nodeId, status }) => {
    const n = nodes.value.find(x => x.id === nodeId)
    if (n) n.status = status
  })
  EventsOn('workflow:status', ({ status }) => {
    store.setRunning(store.selectedScriptID, false)
  })
})

onUnmounted(() => {
  EventsOff('workflow:node-status')
  EventsOff('workflow:status')
})

async function run() {
  nodes.value.forEach(n => n.status = 'idle')
  store.setRunning(store.selectedScriptID, true)
  await RunWorkflow(wfId.value)
}

async function stop() {
  // stop all running scripts in this workflow
  nodes.value.filter(n => n.status === 'running').forEach(n => {
    store.setRunning(n.scriptId, false)
  })
  store.setRunning(store.selectedScriptID, false)
}

</script>

<style scoped>
.wf-run-panel { padding: 8px; }
.config-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.config-header h2 { font-size: 15px; font-weight: 600; color: var(--text); }
.header-actions { display: flex; gap: 6px; align-items: center; }
.badge-running { background: #1b5e20; color: #4caf50; padding: 3px 10px; border-radius: 12px; font-size: 12px; }
.btn-run  { background: #2e7d32; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-stop { background: #c62828; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-edit { background: var(--accent); color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.node-list { display: flex; flex-direction: column; gap: 8px; }
.node-row { display: flex; align-items: center; gap: 10px; padding: 10px 14px; background: var(--surface2); border: 1px solid var(--border); border-radius: 6px; cursor: pointer; }
.node-row:hover { background: var(--surface); }
.node-row.active { border-color: var(--accent); background: var(--surface); }
.dot { width: 10px; height: 10px; border-radius: 50%; background: var(--text-muted); flex-shrink: 0; }
.dot.running { background: #ff9800; }
.dot.success { background: #4caf50; }
.dot.error   { background: #e74c3c; }
.dot.timeout { background: #f57f17; }
.node-name { flex: 1; font-size: 13px; color: var(--text); }
.node-status { font-size: 11px; color: var(--text-muted); }
</style>
