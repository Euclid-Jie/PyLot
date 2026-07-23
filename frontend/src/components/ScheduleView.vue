<template>
  <div class="schedule-view">
    <section class="schedule-card">
      <div class="view-header ui-page-header">
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
          <button class="ui-btn primary" @click="openAdd"><UiIcon name="add" />新增定时</button>
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
              <th>{{ scheduleFilter === 'upcoming' ? '计划运行时间' : '下次运行时间' }}</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredOverview" :key="item.occurrenceKey || item.scheduleId">
              <td>
                <div class="name-cell">
                  <span class="type-dot" :class="item.scriptId < 0 ? 'wf' : 'sc'"></span>
                  <span>{{ item.scriptId < 0 ? (item.scriptName || '工作流') : item.scriptName }}</span>
                </div>
              </td>
              <td><code>{{ item.cronExpr }}</code></td>
              <td>{{ fmtTime(item.nextRun, item.enabled) }}</td>
              <td>
                <button :class="['btn-toggle', item.enabled ? 'on' : 'off']" :disabled="pendingScheduleIds.includes(item.scheduleId)" @click="toggle(item)">
                  {{ pendingScheduleIds.includes(item.scheduleId) ? '处理中' : (item.enabled ? '启用' : '禁用') }}
                </button>
              </td>
              <td>
                <div class="actions">
                  <button class="btn-edit" @click="openEdit(item)">编辑</button>
                  <button class="btn-del" :disabled="pendingScheduleIds.includes(item.scheduleId)" @click="showDeleteId = item.scheduleId">删除</button>
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
          <span>最近 50 条顶层运行</span>
        </div>
        <div class="history-actions">
          <div class="filter-tabs" aria-label="运行来源筛选">
            <button
              v-for="option in historyFilterOptions"
              :key="option.value"
              :class="{ active: historySourceFilter === option.value }"
              @click="changeHistoryFilter(option.value)"
            >
              {{ option.label }}
              <span>{{ option.count }}</span>
            </button>
          </div>
          <button class="btn-edit" :disabled="historyLoading" @click="loadHistory()">
            {{ historyLoading ? '刷新中' : '刷新' }}
          </button>
        </div>
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
            <span class="run-tags">
              <span :class="['type-pill', run.targetType]">{{ targetTypeText(run.targetType) }}</span>
              <span :class="['source-pill', run.triggerSource]" :title="sourceTitle(run)">{{ sourceText(run.triggerSource) }}</span>
            </span>
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
            <div v-if="detailLoading" class="detail-empty">正在加载运行详情...</div>
            <div v-else-if="detailError" class="detail-empty">{{ detailError }}</div>
            <template v-else-if="selectedRun.targetType === 'workflow'">
              <div v-if="workflowNodes.length" class="workflow-detail">
                <label class="workflow-node-select">
                  <span>工作流节点</span>
                  <select :value="selectedWorkflowNode?.nodeId || ''" @change="selectWorkflowNodeById($event.target.value)">
                    <option v-for="node in workflowNodes" :key="node.id" :value="node.nodeId">
                      {{ node.scriptName }} · {{ statusText(node.status) }}
                    </option>
                  </select>
                </label>
                <div class="workflow-node-list" aria-label="工作流节点">
                  <button
                    v-for="node in workflowNodes"
                    :key="node.id"
                    :class="['workflow-node-row', { active: selectedWorkflowNode?.id === node.id }]"
                    @click="selectWorkflowNode(node)"
                  >
                    <span :class="['node-state-dot', node.status]"></span>
                    <span class="workflow-node-main">
                      <strong>{{ node.scriptName || `脚本 #${node.scriptId}` }}</strong>
                      <small>{{ fmtNodeDuration(node) }}</small>
                    </span>
                    <span :class="['workflow-node-status', node.status]">{{ statusText(node.status) }}</span>
                  </button>
                </div>
                <section class="workflow-node-log">
                  <div v-if="selectedWorkflowNode" class="node-log-header">
                    <strong>{{ selectedWorkflowNode.scriptName || `脚本 #${selectedWorkflowNode.scriptId}` }}</strong>
                    <span>{{ statusText(selectedWorkflowNode.status) }} · {{ fmtNodeDuration(selectedWorkflowNode) }}</span>
                  </div>
                  <div v-if="nodeLogLoading" class="detail-empty">正在加载节点日志...</div>
                  <pre v-else-if="detailLog" class="detail-log">{{ detailLog }}</pre>
                  <div v-else class="detail-empty">{{ nodeLogMessage }}</div>
                </section>
              </div>
              <div v-else class="detail-empty">该次运行发生在节点日志记录功能启用前</div>
            </template>
            <template v-else>
              <pre v-if="detailLog" class="detail-log">{{ detailLog }}</pre>
              <div v-else class="detail-empty">该次运行没有日志输出</div>
            </template>
          </template>
        </div>
      </div>
      <div v-else class="empty history-empty">{{ allHistory.length ? '当前筛选下暂无运行记录' : '暂无运行记录' }}</div>
    </section>

    <div v-if="errorMsg" class="toast-error">{{ errorMsg }}</div>
    <TimerModal
      v-if="showTimer"
      :script-id="timerScriptId"
      :schedule="editingSchedule"
      :allow-target-select="true"
      :existing-schedules="overview"
      @saved="handleTimerSaved"
      @close="closeTimer"
    />
    <ConfirmDialog v-if="showDeleteId !== null" title="删除定时任务？" message="该定时规则将被永久删除，此操作无法撤销。" @confirm="del(showDeleteId)" @cancel="showDeleteId = null" />
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { DeleteSchedule, GetRunDetail, GetScheduleOverview, GetTopLevelRunHistory, GetWorkflowRunNodes, GetWorkflowRuns, ToggleSchedule } from '../../wailsjs/go/main/App.js'
import TimerModal from './TimerModal.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import UiIcon from './UiIcon.vue'

const overview = ref([])
const allHistory = ref([])
const selectedRun = ref(null)
const detailLog = ref('')
const detailLoading = ref(false)
const detailError = ref('')
const workflowNodes = ref([])
const selectedWorkflowNode = ref(null)
const nodeLogLoading = ref(false)
const nodeLogMessage = ref('')
const historyLoading = ref(false)
const errorMsg = ref('')
const showTimer = ref(false)
const editingSchedule = ref(null)
const timerScriptId = ref(0)
const scheduleFilter = ref('upcoming')
const historySourceFilter = ref('all')
const showDeleteId = ref(null)
const pendingScheduleIds = ref([])
let timer = null

const UPCOMING_DISPLAY_LIMIT = 50

const sortedOverview = computed(() => [...overview.value].sort(compareSchedule))
const upcomingOverview = computed(() => overview.value
  .filter(item => item.enabled)
  .flatMap(item => (item.upcomingRuns || [])
    .filter(isValidDate)
    .map(nextRun => ({
      ...item,
      nextRun,
      occurrenceKey: `${item.scheduleId}:${new Date(nextRun).getTime()}`,
    })))
  .sort(compareSchedule)
  .slice(0, UPCOMING_DISPLAY_LIMIT))
const filteredOverview = computed(() => scheduleFilter.value === 'upcoming'
  ? upcomingOverview.value
  : sortedOverview.value.filter(matchesScheduleFilter))
const history = computed(() => allHistory.value.filter(run => historySourceFilter.value === 'all' || run.triggerSource === historySourceFilter.value))
const filterOptions = computed(() => [
  { value: 'upcoming', label: '即将运行', count: upcomingOverview.value.length },
  { value: 'today', label: '今日运行', count: overview.value.filter(item => item.enabled && isToday(item.nextRun)).length },
  { value: 'stopped', label: '已停止', count: overview.value.filter(item => !item.enabled).length },
  { value: 'all', label: '全部', count: overview.value.length },
])
const historyFilterOptions = computed(() => [
  { value: 'all', label: '全部', count: allHistory.value.length },
  { value: 'schedule', label: '仅看定时', count: allHistory.value.filter(run => run.triggerSource === 'schedule').length },
  { value: 'manual', label: '仅看手动', count: allHistory.value.filter(run => run.triggerSource === 'manual').length },
])

onMounted(async () => {
  await load()
  timer = setInterval(loadOverview, 60000)
})
onUnmounted(() => clearInterval(timer))

async function load() {
  const items = await loadOverview()
  void loadHistory(items)
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
    allHistory.value = groups
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
      triggerSource: record.triggerSource || 'unknown',
      scheduleId: record.scheduleId || 0,
      logPreview: '',
      hasLog: false,
    }))
  }

  const records = await GetTopLevelRunHistory(item.scriptId) || []
  return records.map(record => ({
    recordId: record.id,
    targetId: item.scriptId,
    targetType: 'script',
    targetName: item.scriptName || `脚本 #${item.scriptId}`,
    status: record.status,
    startedAt: record.startedAt,
    endedAt: record.endedAt,
    isError: record.isError === 1,
    triggerSource: record.triggerSource || 'unknown',
    scheduleId: record.scheduleId || 0,
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

function sourceText(source) {
  const labels = { schedule: '定时', manual: '手动', unknown: '历史' }
  return labels[source] || '历史'
}

function sourceTitle(run) {
  if (run.triggerSource === 'schedule' && run.scheduleId) return `由定时规则 #${run.scheduleId} 触发`
  if (run.triggerSource === 'manual') return '由用户手动触发'
  return '旧版运行记录，来源未记录'
}

async function changeHistoryFilter(source) {
  historySourceFilter.value = source
  const currentKey = selectedRun.value ? runKey(selectedRun.value) : ''
  selectedRun.value = history.value.find(run => runKey(run) === currentKey) || history.value[0] || null
  await loadSelectedDetail()
}

function statusText(status) {
  const labels = {
    pending: '等待中',
    success: '成功',
    error: '失败',
    running: '运行中',
    timeout: '超时',
    killed: '已终止',
    skipped: '未执行',
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
  if (!selectedRun.value || runKey(selectedRun.value) !== runKey(run)) selectedWorkflowNode.value = null
  selectedRun.value = run
  await loadSelectedDetail()
}

async function loadSelectedDetail() {
  const previousNodeID = selectedWorkflowNode.value?.nodeId || ''
  detailLog.value = ''
  detailError.value = ''
  nodeLogMessage.value = ''
  workflowNodes.value = []
  const run = selectedRun.value
  if (!run) return
  detailLoading.value = true
  try {
    if (run.targetType === 'script') {
      const detail = await GetRunDetail(run.recordId)
      detailLog.value = (detail?.logOutput || '').trim()
      selectedWorkflowNode.value = null
      return
    }

    workflowNodes.value = await GetWorkflowRunNodes(run.recordId) || []
    selectedWorkflowNode.value = chooseWorkflowNode(workflowNodes.value, previousNodeID)
    await loadWorkflowNodeLog()
  } catch (err) {
    errorMsg.value = formatError(err)
    detailError.value = '运行详情加载失败'
    setTimeout(() => { errorMsg.value = '' }, 3000)
  } finally {
    detailLoading.value = false
  }
}

function chooseWorkflowNode(nodes, preferredNodeID = '') {
  if (!nodes.length) return null
  const preferred = nodes.find(node => node.nodeId === preferredNodeID)
  if (preferred) return preferred
  const failed = nodes.find(node => ['error', 'timeout', 'killed'].includes(node.status))
  if (failed) return failed
  return [...nodes].reverse().find(node => node.status !== 'pending' && node.status !== 'skipped') || nodes[0]
}

async function selectWorkflowNode(node) {
  selectedWorkflowNode.value = node
  await loadWorkflowNodeLog()
}

async function selectWorkflowNodeById(nodeID) {
  const node = workflowNodes.value.find(item => item.nodeId === nodeID)
  if (node) await selectWorkflowNode(node)
}

async function loadWorkflowNodeLog() {
  detailLog.value = ''
  nodeLogMessage.value = ''
  const node = selectedWorkflowNode.value
  if (!node) return
  if (!node.runRecordId) {
    nodeLogMessage.value = node.status === 'skipped' ? '该节点因前序节点失败而未执行' : '该节点尚未产生运行日志'
    return
  }

  nodeLogLoading.value = true
  try {
    const detail = await GetRunDetail(node.runRecordId)
    if (!detail) {
      nodeLogMessage.value = '该节点的历史日志已被清理'
      return
    }
    detailLog.value = (detail.logOutput || '').trim()
    if (!detailLog.value) nodeLogMessage.value = '该节点没有日志输出'
  } catch (err) {
    errorMsg.value = formatError(err)
    nodeLogMessage.value = '节点日志加载失败'
    setTimeout(() => { errorMsg.value = '' }, 3000)
  } finally {
    nodeLogLoading.value = false
  }
}

function fmtNodeDuration(node) {
  if (!node?.startedAt) return node?.status === 'skipped' ? '未执行' : '-'
  return fmtDuration(node)
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

function closeTimer() {
  showTimer.value = false
  editingSchedule.value = null
  timerScriptId.value = 0
}

async function handleTimerSaved() {
  await loadOverview()
}

async function toggle(item) {
  if (pendingScheduleIds.value.includes(item.scheduleId)) return
  const previous = item.enabled
  item.enabled = !previous
  pendingScheduleIds.value.push(item.scheduleId)
  try {
    errorMsg.value = ''
    await ToggleSchedule(item.scheduleId, item.enabled)
    const refreshed = await GetScheduleOverview() || []
    const latest = refreshed.find(entry => entry.scheduleId === item.scheduleId)
    if (latest) Object.assign(item, latest)
  } catch (err) {
    item.enabled = previous
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  } finally {
    pendingScheduleIds.value = pendingScheduleIds.value.filter(id => id !== item.scheduleId)
  }
}

async function del(id) {
  showDeleteId.value = null
  if (pendingScheduleIds.value.includes(id)) return
  pendingScheduleIds.value.push(id)
  try {
    errorMsg.value = ''
    await DeleteSchedule(id)
    overview.value = overview.value.filter(item => item.scheduleId !== id)
  } catch (err) {
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  } finally {
    pendingScheduleIds.value = pendingScheduleIds.value.filter(scheduleID => scheduleID !== id)
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

.history-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
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
  font-size: var(--type-label);
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
  font-size: var(--type-caption);
  line-height: 1;
  opacity: .8;
}

.view-header h2 {
  font-size: var(--type-page-title);
  font-weight: var(--weight-semibold);
}

.history-header h3 {
  font-size: var(--type-section-title);
  font-weight: var(--weight-semibold);
}

.view-header > div:first-child > span,
.history-header > div:first-child > span {
  display: block;
  margin-top: 2px;
  color: var(--text-muted);
  font-size: var(--type-label);
}

.table-wrap {
  min-height: 0;
  overflow: auto;
}

table {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: var(--type-body);
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
  font-size: var(--type-label);
  font-weight: var(--weight-medium);
  letter-spacing: 0;
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
  font-size: var(--type-body);
  border: 1px solid var(--border);
  font-family: var(--mono);
}

.btn-toggle {
  padding: 4px 14px;
  border-radius: 20px;
  font-size: var(--type-body);
  font-weight: var(--weight-medium);
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
  font-size: var(--type-body);
  transition: color .12s, border-color .12s, background .12s;
}

.btn-add {
  padding: 6px 14px;
  background: var(--accent);
  color: #fff;
  border: 1px solid var(--accent);
  font-weight: var(--weight-medium);
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
  grid-template-columns: 58px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border: none;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  text-align: left;
}

.run-tags {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 3px;
}

.run-row:hover {
  background: var(--surface);
}

.run-row.active {
  background: var(--accent-dim);
}

.type-pill,
.source-pill,
.status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  white-space: nowrap;
  border-radius: 999px;
  font-size: var(--type-caption);
  line-height: 1.6;
}

.type-pill {
  min-height: 19px;
  padding: 0 4px;
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.type-pill.workflow {
  color: var(--orange);
  border-color: rgba(210,153,34,.4);
}

.source-pill {
  min-height: 19px;
  padding: 0 4px;
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.source-pill.schedule {
  border-color: rgba(63,185,80,.34);
  background: var(--green-dim);
  color: var(--green);
}

.source-pill.manual {
  background: var(--surface);
  color: var(--text-dim);
}

.source-pill.unknown {
  border-color: rgba(210,153,34,.32);
  background: var(--orange-dim);
  color: var(--orange);
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
  font-size: var(--type-body);
  font-weight: var(--weight-semibold);
}

.run-main small {
  color: var(--text-muted);
  font-size: var(--type-caption);
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
  font-size: var(--type-body);
  font-weight: var(--weight-semibold);
}

.detail-meta span {
  color: var(--text-muted);
  font-size: var(--type-label);
  white-space: nowrap;
}

.workflow-detail {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(180px, 220px) minmax(0, 1fr);
  margin-top: 10px;
  border-top: 1px solid var(--border);
}

.workflow-node-select {
  display: none;
}

.workflow-node-list {
  min-height: 0;
  overflow-y: auto;
  border-right: 1px solid var(--border);
  background: var(--sidebar-bg);
}

.workflow-node-row {
  width: 100%;
  min-height: 50px;
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 7px 9px;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  text-align: left;
}

.workflow-node-row:hover {
  background: var(--surface-hover);
}

.workflow-node-row.active {
  background: var(--accent-dim);
}

.node-state-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-muted);
}

.node-state-dot.running {
  background: var(--accent);
}

.node-state-dot.success {
  background: var(--green);
}

.node-state-dot.error,
.node-state-dot.timeout,
.node-state-dot.killed {
  background: var(--red);
}

.workflow-node-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.workflow-node-main strong,
.workflow-node-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workflow-node-main strong {
  font-size: var(--type-label);
  font-weight: var(--weight-semibold);
}

.workflow-node-main small,
.workflow-node-status {
  color: var(--text-muted);
  font-size: var(--type-caption);
}

.workflow-node-status.success {
  color: var(--green);
}

.workflow-node-status.running {
  color: var(--accent);
}

.workflow-node-status.error,
.workflow-node-status.timeout,
.workflow-node-status.killed {
  color: var(--red);
}

.workflow-node-log {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding-left: 12px;
}

.node-log-header {
  min-height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 2px;
  border-bottom: 1px solid var(--border);
}

.node-log-header strong {
  min-width: 0;
  overflow: hidden;
  font-size: var(--type-label);
  font-weight: var(--weight-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-log-header span {
  flex-shrink: 0;
  color: var(--text-muted);
  font-size: var(--type-caption);
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
  font-family: var(--mono);
  font-size: var(--type-code);
  line-height: var(--line-code);
  text-align: left;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty,
.detail-empty {
  color: var(--text-muted);
  font-size: var(--type-body);
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
  font-size: var(--type-body);
  box-shadow: 0 8px 24px rgba(0,0,0,.25);
  z-index: 200;
}

@media (max-width: 960px) {
  .schedule-card {
    flex: .75;
  }

  .history-panel {
    flex: 1.25;
  }

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

  .history-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .history-actions {
    width: 100%;
    justify-content: space-between;
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

@media (max-width: 1120px) {
  .workflow-detail {
    display: flex;
    flex-direction: column;
    border-top: 0;
  }

  .workflow-node-select {
    display: grid;
    gap: 5px;
    padding: 10px 0;
    border-bottom: 1px solid var(--border);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  .workflow-node-select select {
    width: 100%;
    min-height: 34px;
    padding: 5px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--input-bg);
    color: var(--text);
  }

  .workflow-node-list {
    display: none;
  }

  .workflow-node-log {
    flex: 1;
    padding-left: 0;
  }
}

@media (max-width: 720px) {
  .history-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .history-actions .btn-edit {
    align-self: flex-end;
  }

  .detail-meta {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }

  .detail-meta span {
    white-space: normal;
  }
}
</style>
