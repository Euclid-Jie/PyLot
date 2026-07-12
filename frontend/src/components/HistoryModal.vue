<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <section class="history-dialog" role="dialog" aria-modal="true" aria-labelledby="history-title">
      <header><div><span>运行记录</span><h3 id="history-title">历史日志</h3></div><button class="ui-icon-btn" title="关闭" aria-label="关闭" @click="$emit('close')"><UiIcon name="x" /></button></header>
      <div class="history-body">
        <aside class="record-list">
          <button v-for="record in records" :key="record.id" :class="['record-item', { active: detail?.id === record.id }]" @click="loadDetail(record.id)">
            <span :class="['status-dot', record.status]"></span>
            <span class="record-main"><strong>{{ statusText(record.status) }}</strong><small>{{ fmt(record.startedAt) }}</small></span>
            <span v-if="record.isError" class="error-label">异常</span>
          </button>
          <div v-if="loading" class="empty">正在加载记录...</div>
          <div v-else-if="!records.length" class="empty">暂无运行记录</div>
        </aside>
        <section class="detail-log">
          <div v-if="detailLoading" class="empty detail-empty">正在加载日志...</div>
          <div v-else-if="detail" class="log-body"><div v-for="(line, index) in detail.logOutput.split('\n')" :key="index" class="log-line">{{ line }}</div></div>
          <div v-else class="empty detail-empty">选择一条记录查看日志</div>
        </section>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { GetRunDetail, GetRunHistory } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import UiIcon from './UiIcon.vue'

defineEmits(['close'])
const store = useMainStore()
const records = ref([])
const detail = ref(null)
const loading = ref(true)
const detailLoading = ref(false)
const labels = { running: '运行中', success: '成功', error: '失败', killed: '已终止', timeout: '已超时' }
onMounted(async () => {
  try {
    if (store.selectedScriptID > 0) records.value = await GetRunHistory(store.selectedScriptID) || []
  } finally {
    loading.value = false
  }
})
async function loadDetail(id) {
  detailLoading.value = true
  try { detail.value = await GetRunDetail(id) }
  finally { detailLoading.value = false }
}
function fmt(time) { return time ? new Date(time).toLocaleString('zh-CN', { hour12: false }) : '' }
function statusText(status) { return labels[status] || status }
</script>

<style scoped>
.history-dialog { width: min(760px, calc(100vw - 40px)); height: min(520px, calc(100vh - 40px)); display: flex; flex-direction: column; overflow: hidden; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--surface-raised); box-shadow: var(--shadow-dialog); }
header { min-height: 66px; display: flex; align-items: flex-start; justify-content: space-between; padding: 16px 18px; border-bottom: 1px solid var(--border); }
header span { color: var(--text-muted); font-size: var(--type-label); }
h3 { font-size: var(--type-object-title); font-weight: var(--weight-semibold); }
.history-body { min-height: 0; flex: 1; display: grid; grid-template-columns: 240px minmax(0, 1fr); }
.record-list { overflow-y: auto; border-right: 1px solid var(--border); background: var(--sidebar-bg); }
.record-item { width: 100%; min-height: 54px; display: grid; grid-template-columns: 8px minmax(0, 1fr) auto; align-items: center; gap: 9px; padding: 8px 12px; border: 0; border-bottom: 1px solid var(--border); background: transparent; color: var(--text); text-align: left; }
.record-item:hover { background: var(--surface-hover); }
.record-item.active { background: var(--accent-dim); }
.status-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-muted); }
.status-dot.success { background: var(--green); } .status-dot.error, .status-dot.killed, .status-dot.timeout { background: var(--red); } .status-dot.running { background: var(--accent); }
.record-main { min-width: 0; display: flex; flex-direction: column; }
.record-main strong { font-size: var(--type-label); font-weight: var(--weight-semibold); } .record-main small { overflow: hidden; color: var(--text-muted); font-size: var(--type-caption); text-overflow: ellipsis; white-space: nowrap; }
.error-label { color: var(--red); font-size: var(--type-caption); }
.detail-log { min-width: 0; min-height: 0; padding: 12px; background: var(--bg); }
.log-body { height: 100%; overflow: auto; padding: 10px 12px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); font-family: var(--mono); font-size: var(--type-code); font-weight: var(--weight-regular); }
.log-line { color: var(--text-dim); line-height: var(--line-code); white-space: pre-wrap; word-break: break-all; }
.empty { padding: 20px; color: var(--text-muted); font-size: var(--type-label); text-align: center; }
.detail-empty { height: 100%; display: grid; place-items: center; }
</style>
