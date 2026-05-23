<template>
  <div class="app-layout">
    <aside class="sidebar"><Sidebar /></aside>
    <main class="main-area">
      <div class="content-area" :class="{ 'no-pad': store.currentView === 'workflow' }">
        <ScheduleView v-if="store.currentView === 'schedule'" />
        <SettingsView v-else-if="store.currentView === 'settings'" />
        <WorkflowEditor v-else-if="store.currentView === 'workflow'" />
        <WorkflowRunPanel v-else-if="store.selectedScriptID !== null && store.selectedScriptID < 0" :key="store.selectedScriptID" />
        <ScriptConfig v-else-if="store.selectedScriptID !== null" :key="store.selectedScriptID" />
        <div v-else class="empty-hint">请从左侧选择脚本，或点击 + 新建</div>
      </div>
      <footer v-if="store.currentView !== 'workflow'" class="log-footer"><LogPanel /></footer>
    </main>
  </div>
  <div v-if="alertData" class="alert-overlay" @click="alertData = null">
    <div class="alert-box" @click.stop>
      <strong>{{ alertData.scriptName }}</strong>
      <p>{{ alertData.reason }}</p>
      <button @click="alertData = null">关闭</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime.js'
import { SetWindowSize, GetWindowSize } from '../wailsjs/go/main/App.js'
import { useMainStore } from './stores/main.js'
import Sidebar from './components/Sidebar.vue'
import ScriptConfig from './components/ScriptConfig.vue'
import LogPanel from './components/LogPanel.vue'
import ScheduleView from './components/ScheduleView.vue'
import WorkflowEditor from './components/WorkflowEditor.vue'
import WorkflowRunPanel from './components/WorkflowRunPanel.vue'
import SettingsView from './components/SettingsView.vue'

const store = useMainStore()
const alertData = ref(null)

onMounted(async () => {
  const theme = localStorage.getItem('theme') || 'dark'
  const font = localStorage.getItem('font') || 'system-ui, sans-serif'
  document.documentElement.setAttribute('data-theme', theme)
  document.documentElement.style.setProperty('--font', font)

  const savedW = parseInt(localStorage.getItem('winW'))
  const savedH = parseInt(localStorage.getItem('winH'))
  if (savedW > 400 && savedH > 300) await SetWindowSize(savedW, savedH)

  window.addEventListener('beforeunload', async () => {
    const [w, h] = await GetWindowSize()
    localStorage.setItem('winW', w)
    localStorage.setItem('winH', h)
  })

  EventsOn('log:line', (d) => {
    if (d.scriptID === store.selectedScriptID) store.addLog(d)
  })
  EventsOn('tray:schedule', () => store.setView('schedule'))
  EventsOn('task:status', (d) => store.setRunning(d.scriptID, d.status === 'running'))
  EventsOn('task:alert', (d) => { alertData.value = d })
})
</script>

<style>
/* CSS variable themes */
:root, [data-theme="dark"] {
  --bg: #1a1a2e; --sidebar-bg: #16213e; --border: #0f3460;
  --surface: #1e2d50; --surface2: #2d2d2d; --input-bg: #252526;
  --text: #e0e0e0; --text-muted: #888; --accent: #1565c0;
  --font: system-ui, sans-serif;
}
[data-theme="light"] {
  --bg: #f5f5f5; --sidebar-bg: #ffffff; --border: #d0d0d0;
  --surface: #ffffff; --surface2: #f0f0f0; --input-bg: #fafafa;
  --text: #1a1a1a; --text-muted: #666; --accent: #1565c0;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: var(--font); background: var(--bg); color: var(--text); height: 100vh; overflow: hidden; }
.app-layout { display: flex; height: 100vh; }
.sidebar { width: 200px; min-width: 200px; background: var(--sidebar-bg); border-right: 1px solid var(--border); overflow-y: auto; display: flex; flex-direction: column; }
.main-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.content-area { flex: 1; overflow-y: auto; padding: 16px; }
.content-area.no-pad { padding: 0; overflow: hidden; }
.log-footer { height: 280px; flex-shrink: 0; border-top: 1px solid var(--border); }
.empty-hint { color: var(--text-muted); text-align: center; margin-top: 80px; font-size: 14px; }
.alert-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 999; }
.alert-box { background: var(--sidebar-bg); border: 1px solid #e74c3c; padding: 24px; border-radius: 8px; min-width: 300px; }
.alert-box strong { color: #e74c3c; font-size: 16px; }
.alert-box p { margin: 10px 0 16px; color: var(--text-muted); font-size: 14px; }
.alert-box button { padding: 6px 16px; background: #e74c3c; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
button { cursor: pointer; }
input, select, textarea { outline: none; }
</style>
