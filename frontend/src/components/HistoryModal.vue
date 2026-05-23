<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>历史运行记录</h3>
        <button @click="$emit('close')">✕</button>
      </div>
      <div class="record-list">
        <div v-for="r in records" :key="r.id" class="record-item" @click="loadDetail(r.id)">
          <span :class="['status', r.status]">{{ r.status }}</span>
          <span class="time">{{ fmt(r.startedAt) }}</span>
          <span class="time">→ {{ r.endedAt ? fmt(r.endedAt) : '—' }}</span>
          <span v-if="r.isError" class="err-badge">异常</span>
        </div>
        <div v-if="!records.length" class="empty">暂无记录</div>
      </div>
      <div v-if="detail" class="detail-log">
        <div class="log-body">
          <div v-for="(line, i) in detail.logOutput.split('\n')" :key="i" class="log-line">{{ line }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GetRunHistory, GetRunDetail } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'

defineEmits(['close'])
const store = useMainStore()
const records = ref([])
const detail = ref(null)

onMounted(async () => {
  if (store.selectedScriptID > 0) {
    records.value = await GetRunHistory(store.selectedScriptID) || []
  }
})

async function loadDetail(id) {
  detail.value = await GetRunDetail(id)
}

function fmt(t) {
  if (!t) return ''
  return new Date(t).toLocaleString()
}
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: #16213e; border: 1px solid #0f3460; border-radius: 8px; padding: 20px; width: 700px; max-height: 80vh; display: flex; flex-direction: column; }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.modal-header button { background: none; border: none; color: #aaa; font-size: 16px; cursor: pointer; }
.record-list { overflow-y: auto; max-height: 200px; border: 1px solid #333; border-radius: 4px; }
.record-item { display: flex; gap: 12px; align-items: center; padding: 7px 12px; font-size: 13px; cursor: pointer; border-bottom: 1px solid #222; }
.record-item:hover { background: #1e2d50; }
.status { padding: 2px 6px; border-radius: 3px; font-size: 11px; font-weight: bold; }
.status.success { background: #1b5e20; color: #4caf50; }
.status.error { background: #4a0000; color: #f44747; }
.status.running { background: #0d47a1; color: #64b5f6; }
.status.killed { background: #3e2723; color: #ff8a65; }
.status.timeout { background: #4a3000; color: #ffb300; }
.time { color: #aaa; font-size: 12px; }
.err-badge { background: #e74c3c; color: #fff; padding: 1px 6px; border-radius: 3px; font-size: 11px; }
.empty { padding: 16px; text-align: center; color: #555; font-size: 13px; }
.detail-log { margin-top: 12px; flex: 1; overflow: hidden; }
.log-body { background: #1e1e1e; border-radius: 4px; padding: 8px 12px; height: 200px; overflow-y: auto; font-family: monospace; font-size: 12px; }
.log-line { line-height: 1.6; color: #d4d4d4; white-space: pre-wrap; word-break: break-all; text-align: left; display: block; }
</style>
