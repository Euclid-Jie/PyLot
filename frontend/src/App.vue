<template>
  <div class="app-layout">
    <aside class="sidebar"><Sidebar /></aside>
    <main class="main-area" :class="{ 'script-mode': store.currentView === 'script', 'detail-open': store.selectedScriptID !== null }">
      <ScriptListPane v-if="store.currentView === 'script'" />
      <section class="workspace-main">
      <div class="content-area" :class="{ 'no-pad': store.currentView === 'workflow' }">
        <ScheduleView v-if="store.currentView === 'schedule'" />
        <SettingsView v-else-if="store.currentView === 'settings'" />
        <ServicesView v-else-if="store.currentView === 'services'" />
        <WorkflowEditor v-else-if="store.currentView === 'workflow'" />
        <WorkflowRunPanel v-else-if="store.selectedScriptID !== null && store.selectedScriptID < 0" :key="store.selectedScriptID" />
        <ScriptConfig v-else-if="store.selectedScriptID !== null" :key="store.selectedScriptID" />
        <div v-else class="empty-state">
          <div class="empty-icon"><UiIcon name="terminal" :size="28" /></div>
          <h1>开始管理 Python 脚本</h1>
          <p>添加脚本后，可直接运行、定时调度，或将多个任务编排成工作流。</p>
          <div class="empty-actions">
            <button class="ui-btn primary" @click="store.setScript(0)"><UiIcon name="add" />添加脚本</button>
            <button class="ui-btn" @click="store.newWorkflow()"><UiIcon name="flow" />新建工作流</button>
          </div>
        </div>
      </div>

      <template v-if="showGlobalLogFooter">
        <button v-if="logCollapsed" class="log-collapsed" title="展开输出面板" @click="expandLog">
          <UiIcon name="terminal" />
          <span>输出</span>
          <strong v-if="store.currentLogs.length">{{ store.currentLogs.length }}</strong>
        </button>
        <footer v-else class="log-footer" :style="{ height: `${logHeight}px` }">
          <div class="resize-handle" @mousedown="startResize"></div>
          <LogPanel @collapse="collapseLog" />
        </footer>
      </template>
      </section>
    </main>
  </div>

  <div v-if="alertData" class="dialog-overlay" @click.self="alertData = null">
    <div class="alert-box" role="alertdialog" aria-modal="true">
      <div class="alert-icon">!</div>
      <div class="alert-content">
        <strong>{{ alertData.scriptName }}</strong>
        <p>{{ alertData.reason }}</p>
      </div>
      <button class="ui-icon-btn" aria-label="关闭" title="关闭" @click="alertData = null"><UiIcon name="x" /></button>
    </div>
  </div>
  <ConfirmDialog v-if="store.navigationBlocked" title="放弃未保存的修改？" message="当前页面有尚未保存的内容。继续离开后，这些修改将会丢失。" confirm-text="放弃修改" @confirm="store.confirmNavigation()" @cancel="store.cancelNavigation()" />
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime.js'
import { GetRunningScripts, GetRunningWorkflows, GetWindowSize, SetWindowSize } from '../wailsjs/go/main/App.js'
import { useMainStore } from './stores/main.js'
import { resolveFont } from './utils/font.js'
import LogPanel from './components/LogPanel.vue'
import ScheduleView from './components/ScheduleView.vue'
import ScriptConfig from './components/ScriptConfig.vue'
import ScriptListPane from './components/ScriptListPane.vue'
import ServicesView from './components/ServicesView.vue'
import SettingsView from './components/SettingsView.vue'
import Sidebar from './components/Sidebar.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import UiIcon from './components/UiIcon.vue'
import WorkflowEditor from './components/WorkflowEditor.vue'
import WorkflowRunPanel from './components/WorkflowRunPanel.vue'

const store = useMainStore()
const alertData = ref(null)
const logHeight = ref(Number(localStorage.getItem('logHeight')) || 240)
const logCollapsed = ref(localStorage.getItem('logCollapsed') === '1')
const showGlobalLogFooter = computed(() => store.currentView === 'script' && store.selectedScriptID > 0)
let resizing = false

function startResize(event) { resizing = true; event.preventDefault() }
function onMouseMove(event) {
  if (!resizing) return
  logHeight.value = Math.min(520, Math.max(140, window.innerHeight - event.clientY))
}
function stopResize() {
  if (!resizing) return
  resizing = false
  localStorage.setItem('logHeight', String(logHeight.value))
}
function collapseLog() { logCollapsed.value = true; localStorage.setItem('logCollapsed', '1') }
function expandLog() { logCollapsed.value = false; localStorage.setItem('logCollapsed', '0') }

async function saveWindowState() {
  const [width, height] = await GetWindowSize()
  localStorage.setItem('winW', width)
  localStorage.setItem('winH', height)
}

onMounted(async () => {
  const theme = localStorage.getItem('theme') || 'dark'
  document.documentElement.setAttribute('data-theme', theme)
  document.documentElement.style.setProperty('--font', resolveFont())

  const savedW = parseInt(localStorage.getItem('winW'))
  const savedH = parseInt(localStorage.getItem('winH'))
  if (savedW > 400 && savedH > 300) await SetWindowSize(savedW, savedH)
  const [runningScripts, runningWorkflows] = await Promise.all([GetRunningScripts(), GetRunningWorkflows()])
  for (const scriptID of runningScripts || []) store.setRunning(scriptID, true)
  for (const workflowID of runningWorkflows || []) store.setRunning(-workflowID, true)

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', stopResize)
  window.addEventListener('beforeunload', saveWindowState)

  EventsOn('log:line', data => { if (data.scriptID === store.selectedScriptID) store.addLog(data) })
  EventsOn('tray:schedule', () => store.setView('schedule'))
  EventsOn('task:status', data => store.setRunning(data.scriptID, data.status === 'running'))
  EventsOn('task:alert', data => { alertData.value = data })
})

onUnmounted(() => {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', stopResize)
  window.removeEventListener('beforeunload', saveWindowState)
})
</script>

<style>
.app-layout { display: flex; height: 100vh; background: var(--bg); }
.sidebar { width: 248px; min-width: 248px; overflow: hidden; border-right: 1px solid var(--border); background: var(--sidebar-bg); }
.main-area { position: relative; min-width: 0; flex: 1; display: flex; overflow: hidden; }
.workspace-main { position: relative; min-width: 0; flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.content-area { min-height: 0; flex: 1; overflow-y: auto; padding: 24px 28px; }
.content-area.no-pad { padding: 0; overflow: hidden; }
.empty-state { min-height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 48px; text-align: center; }
.empty-icon { width: 56px; height: 56px; display: grid; place-items: center; margin-bottom: 18px; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--surface); color: var(--accent); }
.empty-state h1 { margin-bottom: 7px; color: var(--text); font-size: var(--type-page-title); font-weight: var(--weight-semibold); }
.empty-state p { max-width: 430px; color: var(--text-muted); font-size: var(--type-body); }
.empty-actions { display: flex; gap: 8px; margin-top: 22px; }
.log-footer { position: relative; min-height: 140px; flex-shrink: 0; border-top: 1px solid var(--border); }
.resize-handle { position: absolute; z-index: 10; top: -3px; right: 0; left: 0; height: 6px; cursor: ns-resize; }
.resize-handle:hover { background: var(--accent); }
.log-collapsed { position: absolute; right: 14px; bottom: 14px; z-index: 20; min-height: 34px; display: flex; align-items: center; gap: 7px; padding: 0 12px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface-raised); color: var(--text-dim); box-shadow: 0 6px 20px rgba(0, 0, 0, .22); }
.log-collapsed:hover { border-color: var(--border-strong); color: var(--text); }
.log-collapsed strong { min-width: 18px; padding: 0 5px; border-radius: 999px; background: var(--accent-dim); color: var(--accent); font-size: var(--type-caption); }
.alert-box { width: min(460px, calc(100vw - 40px)); display: grid; grid-template-columns: 34px 1fr auto; align-items: flex-start; gap: 12px; padding: 18px; border: 1px solid rgba(248, 81, 73, .38); border-radius: var(--radius-lg); background: var(--surface-raised); box-shadow: var(--shadow-dialog); }
.alert-icon { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 50%; background: var(--red-dim); color: var(--red); font-weight: var(--weight-semibold); }
.alert-content strong { color: var(--text); font-size: var(--type-section-title); font-weight: var(--weight-semibold); }
.alert-content p { margin-top: 5px; color: var(--text-dim); font-size: var(--type-body); }
@media (max-width: 1050px) {
  .sidebar { width: 220px; min-width: 220px; }
  .content-area { padding: 20px; }
  .main-area.script-mode:not(.detail-open) .workspace-main { display: none; }
  .main-area.script-mode:not(.detail-open) .script-list-pane { width: 100%; flex-basis: 100%; border-right: 0; }
  .main-area.script-mode.detail-open .script-list-pane { display: none; }
}
</style>
