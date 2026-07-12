<template>
  <div class="log-panel">
    <div class="log-toolbar">
      <div class="log-heading"><UiIcon name="terminal" /><span>输出</span><strong>{{ filteredLogs.length }}</strong></div>
      <div class="log-tools">
        <label class="log-search"><UiIcon name="search" :size="14" /><input v-model.trim="query" aria-label="搜索日志" placeholder="搜索日志" /></label>
        <button class="ui-btn small" :class="{ active: errorsOnly }" @click="errorsOnly = !errorsOnly">仅看错误</button>
        <button class="ui-icon-btn" :class="{ active: autoFollow }" :title="autoFollow ? '暂停自动滚动' : '开启自动滚动'" :aria-label="autoFollow ? '暂停自动滚动' : '开启自动滚动'" @click="autoFollow = !autoFollow"><UiIcon :name="autoFollow ? 'pause' : 'play'" /></button>
        <button class="ui-icon-btn" title="历史记录" aria-label="历史记录" @click="showHistory = true"><UiIcon name="history" /></button>
        <button class="ui-icon-btn" :disabled="!workDir" title="在 VS Code 中打开工作目录" aria-label="在 VS Code 中打开工作目录" @click="openWorkDir('vscode')"><UiIcon name="terminal" /></button>
        <button class="ui-icon-btn" :disabled="!workDir" title="在文件夹中打开工作目录" aria-label="在文件夹中打开工作目录" @click="openWorkDir('explorer')"><UiIcon name="folderOpen" /></button>
        <button class="ui-icon-btn" title="清空输出" aria-label="清空输出" @click="store.clearLogs()"><UiIcon name="delete" /></button>
        <button class="ui-icon-btn" title="折叠输出面板" aria-label="折叠输出面板" @click="$emit('collapse')"><UiIcon name="chevronDown" /></button>
      </div>
    </div>
    <div v-if="openError" class="log-tool-error" role="alert">{{ openError }}</div>
    <div ref="logBody" class="log-body" @scroll="handleScroll">
      <div v-for="(entry, index) in filteredLogs" :key="index" :class="['log-line', { error: entry.isError }]">
        <span class="log-time">{{ entry.timestamp }}</span><span>{{ entry.line }}</span>
      </div>
      <div v-if="!filteredLogs.length" class="log-empty">{{ store.currentLogs.length ? '没有匹配的日志' : '暂无输出' }}</div>
    </div>
  </div>
  <HistoryModal v-if="showHistory" @close="showHistory = false" />
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { OpenInFileExplorer, OpenInVSCode } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import HistoryModal from './HistoryModal.vue'
import UiIcon from './UiIcon.vue'

defineEmits(['collapse'])
const store = useMainStore()
const logBody = ref(null)
const showHistory = ref(false)
const query = ref('')
const errorsOnly = ref(false)
const autoFollow = ref(true)
const openError = ref('')
const workDir = computed(() => store.selectedScriptWorkDir || '')
const filteredLogs = computed(() => {
  const keyword = query.value.toLocaleLowerCase()
  return store.currentLogs.filter(entry => (!errorsOnly.value || entry.isError) && (!keyword || `${entry.timestamp} ${entry.line}`.toLocaleLowerCase().includes(keyword)))
})

watch(() => store.currentLogs.length, async () => {
  if (!autoFollow.value) return
  await nextTick()
  if (logBody.value) logBody.value.scrollTop = logBody.value.scrollHeight
})

function handleScroll() {
  if (!logBody.value) return
  const distance = logBody.value.scrollHeight - logBody.value.scrollTop - logBody.value.clientHeight
  if (distance > 48) autoFollow.value = false
}
async function openWorkDir(target) {
  if (!workDir.value) return
  openError.value = ''
  try {
    if (target === 'vscode') await OpenInVSCode(workDir.value)
    else await OpenInFileExplorer(workDir.value)
  } catch (error) {
    openError.value = String(error?.message || error || '打开工作目录失败').replace(/^Error:\s*/i, '')
  }
}
</script>

<style scoped>
.log-panel { height: 100%; display: flex; flex-direction: column; background: var(--bg); }
.log-toolbar { min-height: 42px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 5px 10px 5px 14px; border-bottom: 1px solid var(--border); background: var(--sidebar-bg); }
.log-heading, .log-tools { display: flex; align-items: center; }
.log-heading { gap: 7px; color: var(--text-dim); font-size: 12px; font-weight: 600; }
.log-heading strong { min-width: 20px; padding: 0 5px; border-radius: 999px; background: var(--surface-hover); color: var(--text-muted); font-size: 11px; text-align: center; }
.log-tools { gap: 4px; }
.log-search { width: min(180px, 20vw); height: 28px; display: flex; align-items: center; gap: 6px; padding: 0 8px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--text-muted); }
.log-search:focus-within { border-color: var(--accent); }
.log-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--text); font-size: 12px; }
.ui-btn.active, .ui-icon-btn.active { border-color: rgba(59, 130, 246, .35); background: var(--accent-dim); color: var(--accent); }
.log-tool-error { padding: 6px 14px; border-bottom: 1px solid rgba(248, 81, 73, .28); background: var(--red-dim); color: var(--red); font-size: 12px; }
.log-body { min-height: 0; flex: 1; overflow: auto; padding: 8px 14px; font-family: var(--mono); font-size: 12px; line-height: 1.65; }
.log-line { display: grid; grid-template-columns: 70px minmax(0, 1fr); gap: 8px; color: var(--text-dim); }
.log-line span:last-child { white-space: pre-wrap; word-break: break-all; }
.log-time { color: var(--text-muted); user-select: none; }
.log-line.error { color: var(--red); }
.log-empty { padding-top: 10px; color: var(--text-muted); font-size: 12px; }
@media (max-width: 900px) { .log-search { display: none; } .log-tools .ui-btn { display: none; } }
</style>
