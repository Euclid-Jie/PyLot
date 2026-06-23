<template>
  <div class="schedule-view">
    <section class="schedule-card">
      <div class="view-header">
        <div>
          <h2>定时任务总览</h2>
          <span>{{ overview.length }} 个定时任务 · {{ filteredOverview.length }} 个当前显示</span>
        </div>
        <div class="header-actions">
          <div class="filter-tabs" aria-label="定时任务筛选">
            <button
              v-for="option in filterOptions"
              :key="option.value"
              :class="{ active: scheduleFilter === option.value }"
              @click="scheduleFilter = option.value"
            >
              {{ option.label }}
              <span>{{ option.count }}</span>
            </button>
          </div>
          <button class="btn-add" @click="openAdd">新增定时</button>
        </div>
      </div>

      <div v-if="filteredOverview.length" class="table-wrap">
        <table>
          <colgroup>
            <col class="col-target">
            <col class="col-cron">
            <col class="col-next">
            <col class="col-status">
            <col class="col-actions">
          </colgroup>
          <thead>
            <tr>
              <th>目标</th>
              <th>Cron 表达式</th>
              <th>下次运行时间</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredOverview" :key="item.scheduleId">
              <td>
                <div class="name-cell">
                  <span class="type-dot" :class="item.scriptId < 0 ? 'wf' : 'sc'"></span>
                  <span>{{ item.scriptId < 0 ? (item.scriptName || '工作流') : item.scriptName }}</span>
                </div>
              </td>
              <td><code>{{ item.cronExpr }}</code></td>
              <td>{{ fmtTime(item.nextRun, item.enabled) }}</td>
              <td>
                <button :class="['btn-toggle', item.enabled ? 'on' : 'off']" @click="toggle(item)">
                  {{ item.enabled ? '启用' : '禁用' }}
                </button>
              </td>
              <td>
                <div class="actions">
                  <button class="btn-edit" @click="openEdit(item)">编辑</button>
                  <button class="btn-del" @click="del(item.scheduleId)">删除</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty">{{ overview.length ? '当前筛选下暂无定时任务' : '暂无定时任务' }}</div>
    </section>

    <section class="history-panel">
      <div class="history-header">
        <div>
          <h3>最近运行情况</h3>
          <span>最近 50 条</span>
        </div>
        <button class="btn-edit" :disabled="historyLoading" @click="loadHistory()">
          {{ historyLoading ? '刷新中' : '刷新' }}
        </button>
      </div>

      <div v-if="history.length" class="history-layout">
        <div class="run-list">
          <button
            v-for="run in history"
            :key="runKey(run)"
            class="run-row"
            :class="{ active: selectedRun && runKey(selectedRun) === runKey(run) }"
            @click="selectRun(run)"
          >
            <span :class="['type-pill', run.targetType]">{{ targetTypeText(run.targetType) }}</span>
            <span class="run-main">
              <strong>{{ run.targetName }}</strong>
              <small>{{ fmtDateTime(run.startedAt) }} · {{ fmtDuration(run) }}</small>
            </span>
            <span :class="['status-pill', statusClass(run)]">{{ statusText(run.status) }}</span>
          </button>
        </div>

        <div class="run-detail">
          <template v-if="selectedRun">
            <div class="detail-meta">
              <strong>{{ selectedRun.targetName }}</strong>
              <span>{{ fmtDateTime(selectedRun.startedAt) }} - {{ selectedRun.endedAt ? fmtDateTime(selectedRun.endedAt) : '运行中' }}</span>
            </div>
            <pre v-if="detailLog" class="detail-log">{{ detailLog }}</pre>
            <div v-else class="detail-empty">{{ selectedRun.targetType === 'workflow' ? '工作流仅记录整体状态' : '该次运行没有日志输出' }}</div>
          </template>
        </div>
      </div>
      <div v-else class="empty history-empty">暂无运行记录</div>
    </section>

    <div v-if="errorMsg" class="toast-error">{{ errorMsg }}</div>
    <TimerModal
      v-if="showTimer"
      :script-id="timerScriptId"
      :schedule="editingSchedule"
      :allow-target-select="true"
      :existing-schedules="overview"
      @saved="load"
      @close="closeTimer"
    />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { DeleteSchedule, GetRunDetail, GetRunHistory, GetScheduleOverview, GetWorkflowRuns, ToggleSchedule } from '../../wailsjs/go/main/App.js'
import TimerModal from './TimerModal.vue'

const overview = ref([])
const history = ref([])
const selectedRun = ref(null)
const detailLog = ref('')
const historyLoading = ref(false)
const errorMsg = ref('')
const showTimer = ref(false)
const editingSchedule = ref(null)
const timerScriptId = ref(0)
const scheduleFilter = ref('upcoming')
let timer = null

const sortedOverview = computed(() => [...overview.value].sort(compareSchedule))
const filteredOverview = computed(() => sortedOverview.value.filter(matchesScheduleFilter))
const filterOptions = computed(() => [
  { value: 'upcoming', label: '即将运行', count: overview.value.filter(item => item.enabled && isValidDate(item.nextRun)).length },
  { value: 'today', label: '今日运行', count: overview.value.filter(item => item.enabled && isToday(item.nextRun)).length },
  { value: 'stopped', label: '已停止', count: overview.value.filter(item => !item.enabled).length },
  { value: 'all', label: '全部', count: overview.value.length },
])

onMounted(async () => {
  await load()
  timer = setInterval(load, 60000)
})
onUnmounted(() => clearInterval(timer))

async function load() {
  const items = await loadOverview()
  await loadHistory(items)
}

async function loadOverview() {
  overview.value = await GetScheduleOverview() || []
  return overview.value
}

async function loadHistory(sourceOverview = overview.value) {
  historyLoading.value = true
  try {
    const previousKey = selectedRun.value ? runKey(selectedRun.value) : ''
    const uniqueTargets = Array.from(new Map(sourceOverview.map(item => [item.scriptId, item])).values())
    const groups = await Promise.all(uniqueTargets.map(loadTargetHistory))
    history.value = groups
      .flat()
      .sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
      .slice(0, 50)
    selectedRun.value = history.value.find(run => runKey(run) === previousKey) || history.value[0] || null
    await loadSelectedDetail()
  } finally {
    historyLoading.value = false
  }
}

async function loadTargetHistory(item) {
  if (item.scriptId < 0) {
    const workflowId = -item.scriptId
    const records = await GetWorkflowRuns(workflowId) || []
    return records.map(record => ({
      recordId: record.id,
      targetId: workflowId,
      targetType: 'workflow',
      targetName: item.scriptName || '工作流',
      status: record.status,
      startedAt: record.startedAt,
      endedAt: record.endedAt,
      isError: record.status === 'error' || record.status === 'timeout' || record.status === 'killed',
      logPreview: '',
      hasLog: false,
    }))
  }

  const records = await GetRunHistory(item.scriptId) || []
  return records.map(record => ({
    recordId: record.id,
    targetId: item.scriptId,
    targetType: 'script',
    targetName: item.scriptName || `脚本 #${item.scriptId}`,
    status: record.status,
    startedAt: record.startedAt,
    endedAt: record.endedAt,
    isError: record.isError === 1,
    logPreview: '',
    hasLog: true,
  }))
}

function fmtTime(t, enabled) {
  if (!enabled || !t) return '-'
  const dt = new Date(t)
  if (Number.isNaN(dt.getTime()) || dt.getFullYear() <= 1) return '-'
  return dt.toLocaleString()
}

function compareSchedule(a, b) {
  const aTime = scheduleTime(a)
  const bTime = scheduleTime(b)
  if (aTime !== bTime) return aTime - bTime
  if (a.enabled !== b.enabled) return a.enabled ? -1 : 1
  return targetName(a).localeCompare(targetName(b), 'zh-CN')
}

function matchesScheduleFilter(item) {
  if (scheduleFilter.value === 'upcoming') return item.enabled && isValidDate(item.nextRun)
  if (scheduleFilter.value === 'today') return item.enabled && isToday(item.nextRun)
  if (scheduleFilter.value === 'stopped') return !item.enabled
  return true
}

function scheduleTime(item) {
  if (!item.enabled) return Number.POSITIVE_INFINITY
  const dt = new Date(item.nextRun)
  if (Number.isNaN(dt.getTime()) || dt.getFullYear() <= 1) return Number.POSITIVE_INFINITY - 1
  return dt.getTime()
}

function isValidDate(t) {
  if (!t) return false
  const dt = new Date(t)
  return !Number.isNaN(dt.getTime()) && dt.getFullYear() > 1
}

function isToday(t) {
  if (!isValidDate(t)) return false
  const dt = new Date(t)
  const now = new Date()
  return dt.getFullYear() === now.getFullYear() && dt.getMonth() === now.getMonth() && dt.getDate() === now.getDate()
}

function targetName(item) {
  return item.scriptId < 0 ? (item.scriptName || '工作流') : (item.scriptName || '')
}

function fmtDateTime(t) {
  if (!t) return '-'
  const dt = new Date(t)
  if (Number.isNaN(dt.getTime()) || dt.getFullYear() <= 1) return '-'
  return dt.toLocaleString()
}

function fmtDuration(run) {
  if (!run?.startedAt || !run?.endedAt) return '运行中'
  const start = new Date(run.startedAt).getTime()
  const end = new Date(run.endedAt).getTime()
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) return '-'
  const seconds = Math.round((end - start) / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${seconds % 60}s`
}

function targetTypeText(type) {
  return type === 'workflow' ? '工作流' : '脚本'
}

function statusText(status) {
  const labels = {
    success: '成功',
    error: '失败',
    running: '运行中',
    timeout: '超时',
    killed: '已终止',
  }
  return labels[status] || status || '-'
}

function statusClass(run) {
  if (run.isError) return 'error'
  return run.status || 'unknown'
}

function runKey(run) {
  return `${run.targetType}:${run.recordId}`
}

async function selectRun(run) {
  selectedRun.value = run
  await loadSelectedDetail()
}

async function loadSelectedDetail() {
  detailLog.value = ''
  const run = selectedRun.value
  if (!run) return
  if (run.targetType === 'script') {
    const detail = await GetRunDetail(run.recordId)
    detailLog.value = (detail?.logOutput || '').trim()
  } else {
    detailLog.value = (run.logPreview || '').trim()
  }
}

function openAdd() {
  editingSchedule.value = null
  timerScriptId.value = 0
  showTimer.value = true
}

function openEdit(item) {
  editingSchedule.value = {
    id: item.scheduleId,
    scheduleId: item.scheduleId,
    scriptId: item.scriptId,
    cronExpr: item.cronExpr,
    enabled: item.enabled,
  }
  timerScriptId.value = item.scriptId
  showTimer.value = true
}

async function closeTimer() {
  showTimer.value = false
  editingSchedule.value = null
  timerScriptId.value = 0
  await load()
}

async function toggle(item) {
  try {
    errorMsg.value = ''
    await ToggleSchedule(item.scheduleId, !item.enabled)
    await loadOverview()
  } catch (err) {
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  }
}

async function del(id) {
  if (!confirm('确认删除此定时任务？')) return
  try {
    errorMsg.value = ''
    await DeleteSchedule(id)
    await load()
  } catch (err) {
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  }
}

function formatError(err) {
  const message = err?.message || String(err || '操作失败')
  return message.replace(/^Error:\s*/, '')
}
</script>

<style scoped>
.schedule-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.schedule-card,
.history-panel {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.schedule-card {
  flex: 1.05;
}

.history-panel {
  flex: .95;
  min-height: 220px;
}

.view-header,
.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg);
  flex-shrink: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}

.filter-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--sidebar-bg);
}

.filter-tabs button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 4px 9px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-dim);
  font-size: 12px;
  white-space: nowrap;
}

.filter-tabs button:hover {
  color: var(--text);
  background: var(--surface);
}

.filter-tabs button.active {
  color: var(--accent);
  border-color: rgba(47,129,247,.45);
  background: var(--accent-dim);
}

.filter-tabs span {
  margin: 0;
  color: inherit;
  font-size: 11px;
  line-height: 1;
  opacity: .8;
}

.view-header h2,
.history-header h3 {
  font-size: 16px;
  font-weight: 600;
}

.view-header span,
.history-header span {
  display: block;
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
}

.table-wrap {
  min-height: 0;
  overflow: auto;
}

table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 14px;
}

.col-target {
  width: 28%;
}

.col-cron {
  width: 28%;
}

.col-next {
  width: 24%;
}

.col-status {
  width: 10%;
}

.col-actions {
  width: 10%;
}

th {
  position: sticky;
  top: 0;
  z-index: 1;
  text-align: left;
  padding: 8px 14px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: .04em;
  text-transform: uppercase;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
  background: var(--bg);
}

td {
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
  color: var(--text-dim);
}

td:first-child {
  color: var(--text);
}

th:nth-child(2),
td:nth-child(2),
th:nth-child(3),
td:nth-child(3) {
  text-align: left;
}

th:nth-child(4),
td:nth-child(4) {
  text-align: center;
}

.name-cell {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-cell span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-dot {
  width: 2px;
  height: 18px;
  border-radius: 2px;
  flex-shrink: 0;
}

.type-dot.sc {
  background: var(--accent);
}

.type-dot.wf {
  background: var(--orange);
}

code {
  background: var(--surface);
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  color: var(--green);
  font-size: 13px;
  border: 1px solid var(--border);
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
}

.btn-toggle {
  padding: 4px 14px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity .12s;
}

.btn-toggle.on {
  background: var(--orange-dim);
  color: var(--orange);
  border: 1px solid rgba(210,153,34,.4);
}

.btn-toggle.off {
  background: var(--surface2);
  color: var(--text-muted);
  border: 1px solid var(--border);
}

.btn-toggle:hover {
  opacity: .8;
}

.actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.btn-add,
.btn-edit,
.btn-del {
  border-radius: var(--radius);
  font-size: 13px;
  transition: color .12s, border-color .12s, background .12s;
}

.btn-add {
  padding: 6px 14px;
  background: var(--accent);
  color: #fff;
  border: 1px solid var(--accent);
  font-weight: 500;
}

.btn-add:hover {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
}

.btn-edit {
  background: var(--surface2);
  color: var(--text-dim);
  border: 1px solid var(--border);
  padding: 4px 12px;
}

.btn-edit:hover:not(:disabled) {
  color: var(--text);
  border-color: var(--text-muted);
}

.btn-edit:disabled {
  cursor: default;
  opacity: .55;
}

.btn-del {
  background: transparent;
  color: var(--text-muted);
  border: 1px solid var(--border);
  padding: 4px 12px;
}

.btn-del:hover {
  color: var(--red);
  border-color: var(--red);
}

.history-layout {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(280px, 420px) minmax(0, 1fr);
}

.run-list {
  min-height: 0;
  overflow-y: auto;
  border-right: 1px solid var(--border);
  background: var(--sidebar-bg);
}

.run-row {
  width: 100%;
  min-height: 54px;
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border: none;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  text-align: left;
}

.run-row:hover {
  background: var(--surface);
}

.run-row.active {
  background: var(--accent-dim);
}

.type-pill,
.status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  white-space: nowrap;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.6;
}

.type-pill {
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.type-pill.workflow {
  color: var(--orange);
  border-color: rgba(210,153,34,.4);
}

.run-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.run-main strong,
.run-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.run-main strong {
  font-size: 13px;
  font-weight: 600;
}

.run-main small {
  color: var(--text-muted);
  font-size: 11px;
}

.status-pill {
  min-width: 52px;
  padding: 1px 8px;
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.status-pill.success {
  background: var(--green-dim);
  color: var(--green);
  border-color: rgba(63,185,80,.35);
}

.status-pill.error,
.status-pill.timeout,
.status-pill.killed {
  background: var(--red-dim);
  color: var(--red);
  border-color: rgba(248,81,73,.35);
}

.status-pill.running {
  background: var(--accent-dim);
  color: var(--accent);
  border-color: rgba(47,129,247,.35);
}

.run-detail {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: var(--bg);
}

.detail-meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 9px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.detail-meta strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}

.detail-meta span {
  color: var(--text-muted);
  font-size: 12px;
  white-space: nowrap;
}

.detail-log {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  color: var(--text);
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
  text-align: left;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty,
.detail-empty {
  color: var(--text-muted);
  font-size: 13px;
}

.empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.history-empty {
  min-height: 140px;
}

.detail-empty {
  margin-top: 12px;
}

.toast-error {
  position: fixed;
  right: 24px;
  bottom: 24px;
  max-width: 360px;
  background: var(--red-dim);
  color: var(--red);
  border: 1px solid rgba(248,81,73,.32);
  border-radius: var(--radius);
  padding: 9px 12px;
  font-size: 13px;
  box-shadow: 0 8px 24px rgba(0,0,0,.25);
  z-index: 200;
}

@media (max-width: 960px) {
  .view-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
    justify-content: space-between;
  }

  .filter-tabs {
    overflow-x: auto;
    max-width: 100%;
  }

  .history-layout {
    grid-template-columns: 1fr;
  }

  .run-list {
    max-height: 220px;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }
}
</style>
