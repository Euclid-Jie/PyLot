<template>
  <div class="app-layout">
    <aside class="sidebar"><Sidebar /></aside>
    <main class="main-area">
      <div class="content-area" :class="{ 'no-pad': store.currentView === 'workflow' }">
        <ScheduleView v-if="store.currentView === 'schedule'" />
        <SettingsView v-else-if="store.currentView === 'settings'" />
        <ServicesView v-else-if="store.currentView === 'services'" />
        <WorkflowEditor v-else-if="store.currentView === 'workflow'" />
        <WorkflowRunPanel v-else-if="store.selectedScriptID !== null && store.selectedScriptID < 0" :key="store.selectedScriptID" />
        <ScriptConfig v-else-if="store.selectedScriptID !== null" :key="store.selectedScriptID" />
        <div v-else class="empty-hint">请从左侧选择脚本，或点击 + 新建</div>
      </div>
      <footer v-if="showGlobalLogFooter" class="log-footer"><LogPanel /></footer>
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
import { computed, ref, onMounted } from 'vue'
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
import ServicesView from './components/ServicesView.vue'

const store = useMainStore()
const alertData = ref(null)
const showGlobalLogFooter = computed(() => !['workflow', 'services', 'schedule'].includes(store.currentView))

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
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap');

:root, [data-theme="dark"] {
  --bg: #0d1117;
  --sidebar-bg: #161b22;
  --border: #30363d;
  --surface: #21262d;
  --surface2: #30363d;
  --input-bg: #0d1117;
  --text: #e6edf3;
  --text-muted: #6e7681;
  --text-dim: #8b949e;
  --accent: #2f81f7;
  --accent-dim: rgba(47,129,247,.15);
  --accent-hover: #388bfd;
  --green: #3fb950;
  --green-dim: rgba(63,185,80,.15);
  --red: #f85149;
  --red-dim: rgba(248,81,73,.15);
  --orange: #d29922;
  --orange-dim: rgba(210,153,34,.15);
  --purple: #bc8cff;
  --font: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --radius: 6px;
  --radius-sm: 4px;
}
[data-theme="light"] {
  --bg: #ffffff;
  --sidebar-bg: #f6f8fa;
  --border: #d0d7de;
  --surface: #ffffff;
  --surface2: #f6f8fa;
  --input-bg: #ffffff;
  --text: #1f2328;
  --text-muted: #9198a1;
  --text-dim: #656d76;
  --accent: #0969da;
  --accent-dim: rgba(9,105,218,.1);
  --accent-hover: #0860ca;
  --green: #1a7f37;
  --green-dim: rgba(26,127,55,.1);
  --red: #cf222e;
  --red-dim: rgba(207,34,46,.1);
  --orange: #9a6700;
  --orange-dim: rgba(154,103,0,.1);
  --purple: #8250df;
}

* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: var(--font);
  font-size: 14px;
  line-height: 1.5;
  background: var(--bg);
  color: var(--text);
  height: 100vh;
  overflow: hidden;
  -webkit-font-smoothing: antialiased;
}
.app-layout { display: flex; height: 100vh; }
.sidebar {
  width: 220px; min-width: 220px;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
  overflow-y: auto; display: flex; flex-direction: column;
}
.main-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.content-area { flex: 1; overflow-y: auto; padding: 24px 28px; }
.content-area.no-pad { padding: 0; overflow: hidden; }
.log-footer { height: 260px; flex-shrink: 0; border-top: 1px solid var(--border); }
.empty-hint {
  color: var(--text-muted); text-align: center; margin-top: 120px;
  font-size: 14px;
}
.alert-overlay { position: fixed; inset: 0; background: rgba(1,4,9,.8); display: flex; align-items: center; justify-content: center; z-index: 999; }
.alert-box {
  background: var(--sidebar-bg); border: 1px solid var(--red);
  padding: 24px; border-radius: var(--radius); min-width: 320px;
  box-shadow: 0 8px 24px rgba(1,4,9,.5);
}
.alert-box strong { color: var(--red); font-size: 15px; font-weight: 600; }
.alert-box p { margin: 10px 0 16px; color: var(--text-dim); font-size: 14px; }
.alert-box button {
  padding: 5px 16px; background: var(--red); color: #fff;
  border: none; border-radius: var(--radius); cursor: pointer; font-size: 14px; font-weight: 500;
}
button { cursor: pointer; font-family: var(--font); }
input, select, textarea { outline: none; font-family: var(--font); }
</style>
