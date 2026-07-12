<template>
  <div class="wf-run-panel">
    <div class="config-header ui-page-header">
      <div><h2>{{ wf?.name || '工作流' }}</h2><p class="ui-page-subtitle">查看节点状态并运行完整流程</p></div>
      <div class="header-actions">
        <span v-if="isRunning" class="ui-status running">运行中</span>
        <button v-if="!isRunning" class="ui-btn primary" :disabled="!!actionPending" @click="run"><UiIcon name="play" />{{ actionPending === 'run' ? '启动中' : '运行' }}</button>
        <button v-else class="ui-btn danger" :disabled="!!actionPending" @click="stop"><UiIcon name="stop" />{{ actionPending === 'stop' ? '停止中' : '停止' }}</button>
        <button class="ui-btn" @click="store.setView('workflow')">编辑工作流</button>
      </div>
    </div>
    <div v-if="actionError" class="action-error">{{ actionError }}</div>
    <div class="node-list">
      <div v-for="node in nodes" :key="node.id" class="node-row"
           :class="{ active: store.selectedScriptID === node.scriptId }"
           @click="store.setScriptFromWorkflow(node.scriptId)">
        <span :class="['dot', node.status]"></span>
        <span class="node-name">{{ node.name }}</span>
        <span class="node-status">{{ statusLabel(node.status) }}</span>
      </div>
    </div>
    <section class="workflow-log">
      <header><span>工作流输出</span><strong>{{ workflowLogs.length }}</strong></header>
      <div class="workflow-log-body">
        <div v-for="(entry, index) in workflowLogs" :key="index" :class="['workflow-log-line', { error: entry.isError }]">
          <span>{{ entry.timestamp }}</span><strong>{{ entry.nodeName }}</strong><code>{{ entry.line }}</code>
        </div>
        <div v-if="!workflowLogs.length" class="workflow-log-empty">暂无输出</div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'
import { GetWorkflows, RunWorkflow, StopWorkflow, GetScripts } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import { statusLabel } from '../utils/status.js'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const wfId = computed(() => -store.selectedScriptID)
const wf = ref(null)
const nodes = ref([])
const isRunning = computed(() => store.runningScripts.has(store.selectedScriptID))
const actionPending = ref('')
const actionError = ref('')
const workflowLogs = ref([])

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
  EventsOn('workflow:node-status', ({ workflowId, nodeId, status }) => {
    if (workflowId !== wfId.value) return
    const n = nodes.value.find(x => x.id === nodeId)
    if (n) n.status = status
  })
  EventsOn('workflow:status', ({ workflowId }) => {
    if (workflowId !== wfId.value) return
    store.setRunning(store.selectedScriptID, false)
  })
  EventsOn('workflow:log', entry => {
    if (entry.workflowId !== wfId.value) return
    const node = nodes.value.find(item => item.scriptId === entry.scriptID)
    workflowLogs.value.push({ ...entry, nodeName: node?.name || `#${entry.scriptID}` })
    if (workflowLogs.value.length > 1000) workflowLogs.value = workflowLogs.value.slice(-1000)
  })
})

onUnmounted(() => {
  EventsOff('workflow:node-status')
  EventsOff('workflow:status')
  EventsOff('workflow:log')
})

async function run() {
  if (actionPending.value) return
  actionPending.value = 'run'
  actionError.value = ''
  workflowLogs.value = []
  nodes.value.forEach(n => n.status = 'idle')
  store.setRunning(store.selectedScriptID, true)
  try {
    await RunWorkflow(wfId.value)
  } catch (error) {
    store.setRunning(store.selectedScriptID, false)
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function stop() {
  if (actionPending.value) return
  actionPending.value = 'stop'
  actionError.value = ''
  try {
    await StopWorkflow(wfId.value)
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

function normalizeError(error) {
  return String(error?.message || error || '操作失败').replace(/^Error:\s*/i, '')
}

</script>

<style scoped>
.wf-run-panel { padding: 8px; }
.config-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.config-header h2 { font-size: 15px; font-weight: 600; color: var(--text); }
.header-actions { display: flex; gap: 6px; align-items: center; }
.node-list { display: flex; flex-direction: column; gap: 8px; }
.node-row { display: flex; align-items: center; gap: 10px; padding: 10px 14px; background: var(--surface2); border: 1px solid var(--border); border-radius: 6px; cursor: pointer; }
.node-row:hover { background: var(--surface); }
.node-row.active { border-color: var(--accent); background: var(--surface); }
.dot { width: 10px; height: 10px; border-radius: 50%; background: var(--text-muted); flex-shrink: 0; }
.node-name { flex: 1; font-size: 13px; color: var(--text); }
.node-status { font-size: 11px; color: var(--text-muted); }
.wf-run-panel { max-width: 980px; margin: 0 auto; padding: 0; }
.config-header { margin-bottom: 20px; }
.config-header h2 { font-size: 19px; }
.action-error { margin: -10px 0 14px; padding: 8px 10px; border: 1px solid rgba(248, 81, 73, .28); border-radius: var(--radius); background: var(--red-dim); color: var(--red); font-size: 12px; }
.node-list { gap: 6px; }
.node-row { min-height: 44px; padding: 8px 12px; border-radius: var(--radius); background: var(--surface); }
.node-row:hover { background: var(--surface-hover); }
.dot.running { background: var(--orange); } .dot.success { background: var(--green); } .dot.error, .dot.timeout { background: var(--red); }
.node-status { min-width: 56px; font-size: 12px; text-align: right; }
.workflow-log { height: 260px; display: flex; flex-direction: column; margin-top: 14px; overflow: hidden; border: 1px solid var(--border); border-radius: var(--radius); }
.workflow-log header { min-height: 36px; display: flex; align-items: center; justify-content: space-between; padding: 0 12px; border-bottom: 1px solid var(--border); background: var(--surface); color: var(--text-dim); font-size: 12px; font-weight: 600; }
.workflow-log header strong { color: var(--text-muted); font-size: 11px; font-weight: 500; }
.workflow-log-body { min-height: 0; flex: 1; overflow: auto; padding: 8px 10px; background: var(--input-bg); font-family: var(--mono); font-size: 12px; }
.workflow-log-line { display: grid; grid-template-columns: 66px 120px minmax(0, 1fr); gap: 8px; color: var(--text-dim); line-height: 1.65; }
.workflow-log-line > span { color: var(--text-muted); }
.workflow-log-line > strong { overflow: hidden; color: var(--accent); font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.workflow-log-line code { color: inherit; white-space: pre-wrap; word-break: break-all; }
.workflow-log-line.error { color: var(--red); }
.workflow-log-empty { padding: 8px 2px; color: var(--text-muted); }
</style>
