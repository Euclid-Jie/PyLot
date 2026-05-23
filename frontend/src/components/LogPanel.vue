<template>
  <div class="log-panel">
    <div class="log-toolbar">
      <span class="log-title">日志输出</span>
      <div class="log-actions">
        <button @click="store.clearLogs()">清空</button>
        <button @click="showHistory = true">历史记录</button>
        <button class="btn-vscode" :disabled="!workDir" @click="openVSCode" title="用 VSCode 打开工作目录">
          <svg width="14" height="14" viewBox="0 0 100 100" fill="currentColor"><path d="M74.9 7.4L40.6 38.5 17 21.6 7.4 26.9l21.2 23.1L7.4 73.1 17 78.4l23.6-16.9 34.3 31.1 18.7-9.3V16.7L74.9 7.4zm9.3 57.4l-22-17.8 22-17.8v35.6z"/></svg>
        </button>
      </div>
    </div>
    <div class="log-body" ref="logBody">
      <div
        v-for="(entry, i) in store.currentLogs"
        :key="i"
        :class="['log-line', { error: entry.isError }]"
      >[{{ entry.timestamp }}] {{ entry.line }}</div>
      <div v-if="!store.currentLogs.length" class="log-empty">暂无日志</div>
    </div>
  </div>
  <HistoryModal v-if="showHistory" @close="showHistory = false" />
</template>

<script setup>
import { ref, watch, nextTick, computed } from 'vue'
import { OpenInVSCode } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import HistoryModal from './HistoryModal.vue'

const store = useMainStore()
const logBody = ref(null)
const showHistory = ref(false)

watch(() => store.currentLogs.length, async () => {
  await nextTick()
  if (logBody.value) logBody.value.scrollTop = logBody.value.scrollHeight
})

// get workDir of selected script from store (passed via prop or store)
const workDir = computed(() => store.selectedScriptWorkDir || '')

async function openVSCode() {
  if (workDir.value) await OpenInVSCode(workDir.value)
}
</script>

<style scoped>
.log-panel { display: flex; flex-direction: column; height: 100%; background: #1e1e1e; }
.log-toolbar { display: flex; justify-content: space-between; align-items: center; padding: 4px 12px; background: #252526; border-bottom: 1px solid #333; flex-shrink: 0; }
.log-title { font-size: 12px; color: #aaa; }
.log-actions { display: flex; gap: 6px; }
.log-actions button { padding: 2px 8px; background: #3a3a3a; color: #ccc; border: 1px solid #555; border-radius: 3px; font-size: 12px; }
.btn-vscode { display: flex; align-items: center; justify-content: center; width: 24px; padding: 2px 4px !important; color: #007acc !important; }
.btn-vscode:disabled { opacity: 0.3; cursor: default; }
.log-body { flex: 1; overflow-y: auto; padding: 6px 12px; font-family: monospace; font-size: 12px; }
.log-line { line-height: 1.6; white-space: pre-wrap; word-break: break-all; color: #d4d4d4; text-align: left; display: block; }
.log-line.error { color: #f44747; }
.log-empty { color: #555; font-size: 12px; margin-top: 8px; }
</style>
