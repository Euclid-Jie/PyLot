<template>
  <div class="app-layout">
    <aside class="sidebar"><Sidebar /></aside>
    <main class="main-area">
      <div class="content-area">
        <ScheduleView v-if="store.currentView === 'schedule'" />
        <ScriptConfig v-else-if="store.selectedScriptID !== null" :key="store.selectedScriptID" />
        <div v-else class="empty-hint">请从左侧选择脚本，或点击 + 新建</div>
      </div>
      <footer class="log-footer"><LogPanel /></footer>
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
import { useMainStore } from './stores/main.js'
import Sidebar from './components/Sidebar.vue'
import ScriptConfig from './components/ScriptConfig.vue'
import LogPanel from './components/LogPanel.vue'
import ScheduleView from './components/ScheduleView.vue'

const store = useMainStore()
const alertData = ref(null)

onMounted(() => {
  EventsOn('log:line', (d) => { if (d.scriptID === store.selectedScriptID) store.addLog(d) })
  EventsOn('task:status', (d) => store.setRunning(d.scriptID, d.status === 'running'))
  EventsOn('task:alert', (d) => { alertData.value = d })
})
</script>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, sans-serif; background: #1a1a2e; color: #e0e0e0; height: 100vh; overflow: hidden; }
.app-layout { display: flex; height: 100vh; }
.sidebar { width: 200px; min-width: 200px; background: #16213e; border-right: 1px solid #0f3460; overflow-y: auto; display: flex; flex-direction: column; }
.main-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.content-area { flex: 1; overflow-y: auto; padding: 16px; }
.log-footer { height: 280px; flex-shrink: 0; border-top: 1px solid #0f3460; }
.empty-hint { color: #666; text-align: center; margin-top: 80px; font-size: 14px; }
.alert-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 999; }
.alert-box { background: #16213e; border: 1px solid #e74c3c; padding: 24px; border-radius: 8px; min-width: 300px; }
.alert-box strong { color: #e74c3c; font-size: 16px; }
.alert-box p { margin: 10px 0 16px; color: #ccc; font-size: 14px; }
.alert-box button { padding: 6px 16px; background: #e74c3c; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
button { cursor: pointer; }
input, select, textarea { outline: none; }
</style>
