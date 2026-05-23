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
.log-panel { display: flex; flex-direction: column; height: 100%; background: var(--bg); }
.log-toolbar { display: flex; justify-content: space-between; align-items: center; padding: 6px 16px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border); flex-shrink: 0; }
.log-title { font-size: 12px; font-weight: 600; letter-spacing: .04em; text-transform: uppercase; color: var(--text-muted); }
.log-actions { display: flex; gap: 6px; }
.log-actions button { padding: 4px 10px; background: transparent; color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; font-weight: 500; transition: background .12s, color .12s; }
.log-actions button:hover { background: var(--surface2); color: var(--text); border-color: var(--text-muted); }
.btn-vscode { display: flex; align-items: center; justify-content: center; width: 28px; padding: 4px !important; color: #4f9ef8 !important; }
.btn-vscode:disabled { opacity: 0.25; cursor: default; }
.log-body { flex: 1; overflow-y: auto; padding: 10px 16px; font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; font-size: 13px; line-height: 1.7; }
.log-line { white-space: pre-wrap; word-break: break-all; color: #c9d1d9; display: block; }
.log-line.error { color: var(--red); }
.log-empty { color: var(--text-muted); font-size: 13px; margin-top: 10px; }
</style>
