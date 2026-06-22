<template>
  <div class="services-view">
    <div class="sv-header">
      <div>
        <h2>服务管理</h2>
        <span class="sv-subtitle">{{ services.length }} 个服务</span>
      </div>
      <button class="btn-primary" @click="openAdd">+ 新增服务</button>
    </div>

    <div class="services-shell">
      <aside class="service-list-pane">
        <button
          v-for="s in services"
          :key="s.id"
          class="service-row"
          :class="{ active: selectedId === s.id }"
          @click="select(s)"
        >
          <span :class="['status-dot', statusClass(s)]"></span>
          <span class="service-row-main">
            <strong>{{ s.name }}</strong>
            <small>{{ s.command }}</small>
          </span>
          <span class="row-side">
            <span v-if="s.auto_start" class="auto-chip">自启</span>
            <span class="mini-status">{{ statusLabel(s) }}</span>
          </span>
        </button>
        <div v-if="!services.length" class="empty-list">暂无服务</div>
      </aside>

      <section class="service-detail-pane">
        <div v-if="showForm" class="service-form">
          <div class="detail-title">
            <h3>{{ editId ? '编辑服务' : '新增服务' }}</h3>
            <button class="btn-ghost btn-xs" @click="closeForm">取消</button>
          </div>

          <label class="field">
            <span>名称</span>
            <input v-model.trim="form.name" class="inp" placeholder="服务名称" />
          </label>

          <label class="field">
            <span>命令</span>
            <input v-model.trim="form.command" class="inp" placeholder='.venv\Scripts\python.exe main.py' />
          </label>

          <label class="field">
            <span>工作目录</span>
            <div class="input-action">
              <input v-model.trim="form.workDir" class="inp" placeholder="C:\Projects\demo" />
              <button class="btn-ghost" @click="chooseWorkDir">选择</button>
            </div>
          </label>

          <label class="toggle-row">
            <input type="checkbox" v-model="form.autoStart" />
            <span>PyLot 启动时自动运行</span>
          </label>

          <div v-if="formError" class="form-error">{{ formError }}</div>
          <div class="form-actions">
            <button class="btn-primary" @click="submitForm">{{ editId ? '保存' : '创建' }}</button>
            <button class="btn-ghost" @click="closeForm">取消</button>
          </div>
        </div>

        <div v-else-if="selected" class="detail-content">
          <div class="detail-toolbar">
            <div class="detail-title">
              <h3>{{ selected.name }}</h3>
              <span :class="['badge', statusClass(selected)]">{{ statusLabel(selected) }}</span>
            </div>
            <div class="toolbar-actions">
              <button v-if="canStart(selected)" class="btn-sm btn-green" @click="start(selected)">启动</button>
              <button v-else class="btn-sm btn-red" :disabled="!selected.running" @click="stop(selected.id)">停止</button>
              <button class="btn-sm" :disabled="selected.status === 'starting' || selected.status === 'stopping'" @click="restart(selected.id)">重启</button>
              <button class="btn-sm" @click="openEdit(selected)">编辑</button>
              <button class="btn-sm btn-red" @click="del(selected.id)">删除</button>
            </div>
          </div>

          <div v-if="actionError" class="action-error">{{ actionError }}</div>

          <div class="meta-grid">
            <div>
              <span>命令</span>
              <code>{{ selected.command }}</code>
            </div>
            <div>
              <span>工作目录</span>
              <code>{{ selected.work_dir || '未设置' }}</code>
            </div>
            <div>
              <span>PID</span>
              <strong>{{ selected.pid || '-' }}</strong>
            </div>
            <div>
              <span>跟随 PyLot 启动</span>
              <label class="switch-row">
                <input
                  type="checkbox"
                  :checked="selected.auto_start"
                  @change="toggleAutoStart(selected, $event.target.checked)"
                />
                <strong>{{ selected.auto_start ? '开启' : '关闭' }}</strong>
              </label>
            </div>
            <div>
              <span>启动时间</span>
              <strong>{{ selected.started_at || '-' }}</strong>
            </div>
            <div>
              <span>停止时间</span>
              <strong>{{ selected.stopped_at || '-' }}</strong>
            </div>
            <div v-if="selected.last_error" class="meta-wide meta-error">
              <span>最近错误</span>
              <code>{{ selected.last_error }}</code>
            </div>
          </div>

          <div class="log-panel">
            <div class="log-panel-header">
              <span>输出日志</span>
              <button class="btn-ghost btn-xs" @click="clearSelectedLogs">清空</button>
            </div>
            <div class="log-lines" ref="logEl">
              <div
                v-for="(l, i) in currentLogs"
                :key="i"
                :class="['log-line', { 'log-err': l.isError }]"
              >
                <span class="log-time">{{ l.timestamp }}</span>
                <span>{{ l.line }}</span>
              </div>
              <div v-if="!currentLogs.length" class="log-empty">暂无输出</div>
            </div>
          </div>
        </div>

        <div v-else class="empty-detail">
          <h3>暂无服务</h3>
          <button class="btn-primary" @click="openAdd">新增服务</button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime.js'
import {
  AddService,
  ClearServiceLogs,
  DeleteService,
  GetServiceLogs,
  ListServices,
  OpenDirectoryDialog,
  RestartService,
  SetServiceAutoStart,
  StartService,
  StopService,
  UpdateService,
} from '../../wailsjs/go/main/App.js'

const services = ref([])
const selectedId = ref(null)
const showForm = ref(false)
const editId = ref(null)
const form = reactive({ name: '', command: '', workDir: '', autoStart: false })
const formError = ref('')
const actionError = ref('')
const logs = reactive({})
const logEl = ref(null)

const selected = computed(() => services.value.find(s => s.id === selectedId.value) || null)
const currentLogs = computed(() => selectedId.value === null ? [] : (logs[selectedId.value] || []))

const labels = {
  stopped: '已停止',
  starting: '启动中',
  running: '运行中',
  stopping: '停止中',
  exited: '已退出',
  failed: '异常退出',
}

async function load() {
  services.value = await ListServices() || []
  if (selectedId.value !== null && !services.value.some(s => s.id === selectedId.value)) {
    selectedId.value = services.value[0]?.id ?? null
  }
  if (selectedId.value === null && services.value.length) {
    selectedId.value = services.value[0].id
  }
  if (selectedId.value !== null) await loadLogs(selectedId.value)
}

async function loadLogs(id) {
  logs[id] = await GetServiceLogs(id) || []
  scrollLog()
}

function openAdd() {
  editId.value = null
  Object.assign(form, { name: '', command: '', workDir: '', autoStart: false })
  formError.value = ''
  actionError.value = ''
  showForm.value = true
}

function openEdit(s) {
  editId.value = s.id
  Object.assign(form, {
    name: s.name,
    command: s.command,
    workDir: s.work_dir,
    autoStart: s.auto_start,
  })
  formError.value = ''
  actionError.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editId.value = null
  formError.value = ''
}

async function chooseWorkDir() {
  const dir = await OpenDirectoryDialog('选择服务工作目录')
  if (dir) form.workDir = dir
}

async function submitForm() {
  if (!form.name || !form.command) {
    formError.value = '名称和命令不能为空'
    return
  }

  try {
    const targetName = form.name
    const targetId = editId.value
    if (targetId) {
      await UpdateService(targetId, form.name, form.command, form.workDir, form.autoStart)
    } else {
      await AddService(form.name, form.command, form.workDir, form.autoStart)
    }
    closeForm()
    await load()
    if (targetId) selectedId.value = targetId
    else selectedId.value = services.value.find(s => s.name === targetName)?.id ?? selectedId.value
    if (selectedId.value !== null) await loadLogs(selectedId.value)
  } catch (e) {
    formError.value = normalizeError(e)
  }
}

async function start(s) {
  actionError.value = ''
  selectedId.value = s.id
  ensureLogs(s.id)
  try {
    await StartService(s.id)
    await load()
  } catch (e) {
    appendLocalLog(s.id, `启动失败: ${normalizeError(e)}`, true)
    actionError.value = normalizeError(e)
  }
}

async function stop(id) {
  actionError.value = ''
  try {
    await StopService(id)
  } catch (e) {
    actionError.value = normalizeError(e)
  } finally {
    setTimeout(load, 300)
  }
}

async function restart(id) {
  actionError.value = ''
  try {
    await RestartService(id)
  } catch (e) {
    appendLocalLog(id, `重启失败: ${normalizeError(e)}`, true)
    actionError.value = normalizeError(e)
  } finally {
    setTimeout(load, 500)
  }
}

async function del(id) {
  if (!confirm('确认删除此服务？')) return
  await DeleteService(id)
  delete logs[id]
  if (selectedId.value === id) selectedId.value = null
  await load()
}

async function select(s) {
  selectedId.value = s.id
  showForm.value = false
  actionError.value = ''
  await loadLogs(s.id)
}

async function clearSelectedLogs() {
  if (selectedId.value === null) return
  await ClearServiceLogs(selectedId.value)
  logs[selectedId.value] = []
}

async function toggleAutoStart(s, enabled) {
  const previous = s.auto_start
  s.auto_start = enabled
  actionError.value = ''
  try {
    await SetServiceAutoStart(s.id, enabled)
  } catch (e) {
    s.auto_start = previous
    actionError.value = normalizeError(e)
  }
}

function ensureLogs(id) {
  if (!logs[id]) logs[id] = []
}

function appendLocalLog(id, line, isError) {
  ensureLogs(id)
  logs[id].push({ line, isError, timestamp: new Date().toLocaleTimeString('zh-CN', { hour12: false }) })
  trimLogs(id)
  scrollLog()
}

function trimLogs(id) {
  if (logs[id]?.length > 1000) logs[id] = logs[id].slice(-1000)
}

function statusLabel(s) {
  return labels[s?.status] || (s?.running ? '运行中' : '已停止')
}

function statusClass(s) {
  if (!s) return 'status-stopped'
  return `status-${s.status || (s.running ? 'running' : 'stopped')}`
}

function canStart(s) {
  return s && !s.running && s.status !== 'starting' && s.status !== 'stopping'
}

function normalizeError(e) {
  return String(e || '').replace(/^Error:\s*/i, '')
}

function applyStatus(d) {
  const s = services.value.find(item => item.id === d.id)
  if (!s) return
  Object.assign(s, {
    running: d.running,
    status: d.status,
    pid: d.pid,
    started_at: d.started_at,
    stopped_at: d.stopped_at,
    exit_code: d.exit_code,
    last_error: d.last_error,
  })
}

function scrollLog() {
  nextTick(() => {
    if (logEl.value) logEl.value.scrollTop = logEl.value.scrollHeight
  })
}

onMounted(() => {
  load()
  EventsOn('service:log', (d) => {
    ensureLogs(d.id)
    logs[d.id].push({ line: d.line, isError: d.isError, timestamp: d.timestamp })
    trimLogs(d.id)
    if (selectedId.value === d.id) scrollLog()
  })
  EventsOn('service:status', applyStatus)
})

onUnmounted(() => {
  EventsOff('service:log')
  EventsOff('service:status')
})
</script>

<style scoped>
.services-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.sv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.sv-header h2 {
  font-size: 16px;
  font-weight: 600;
}

.sv-subtitle {
  display: block;
  margin-top: 2px;
  color: var(--text-muted);
  font-size: 12px;
}

.services-shell {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.service-list-pane {
  min-width: 0;
  overflow-y: auto;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
}

.service-row {
  width: 100%;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border: none;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  text-align: left;
}

.service-row:hover {
  background: var(--surface);
}

.service-row.active {
  background: var(--accent-dim);
}

.service-row-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.service-row-main strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
}

.service-row-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 11px;
}

.mini-status {
  color: var(--text-muted);
  font-size: 11px;
  white-space: nowrap;
}

.row-side {
  display: flex;
  align-items: flex-end;
  flex-direction: column;
  gap: 4px;
}

.auto-chip {
  padding: 1px 6px;
  border: 1px solid var(--accent);
  border-radius: 999px;
  color: var(--accent);
  font-size: 10px;
  line-height: 1.4;
  white-space: nowrap;
}

.service-detail-pane {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}

.detail-content,
.service-form {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
}

.detail-toolbar,
.detail-title,
.toolbar-actions,
.form-actions,
.input-action {
  display: flex;
  align-items: center;
}

.detail-toolbar {
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.detail-title {
  min-width: 0;
  gap: 10px;
}

.detail-title h3 {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 600;
}

.toolbar-actions,
.form-actions {
  gap: 8px;
  flex-shrink: 0;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  flex-shrink: 0;
}

.meta-grid > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
}

.meta-grid span,
.field span,
.toggle-row {
  color: var(--text-muted);
  font-size: 12px;
}

.meta-grid strong,
.meta-grid code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-dim);
  font-size: 12px;
}

.meta-grid code {
  font-family: Consolas, 'Courier New', monospace;
}

.meta-wide {
  grid-column: 1 / -1;
}

.meta-error code {
  color: var(--red);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-action {
  gap: 8px;
}

.input-action .inp {
  flex: 1;
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.switch-row strong {
  color: var(--text-dim);
  font-size: 12px;
}

.inp {
  width: 100%;
  min-width: 0;
  background: var(--input-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 7px 10px;
  color: var(--text);
  font-size: 13px;
}

.inp:focus {
  border-color: var(--accent);
}

.form-error,
.action-error {
  color: var(--red);
  font-size: 12px;
}

.log-panel {
  flex: 1;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.log-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 13px;
  flex-shrink: 0;
}

.log-lines {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  background: var(--bg);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
}

.log-line {
  display: grid;
  grid-template-columns: 62px minmax(0, 1fr);
  gap: 8px;
  line-height: 1.6;
  color: var(--text);
  text-align: left;
}

.log-line span:last-child {
  white-space: pre-wrap;
  word-break: break-all;
}

.log-time {
  color: var(--text-muted);
}

.log-err {
  color: var(--red);
}

.log-empty,
.empty-list,
.empty-detail {
  color: var(--text-muted);
  font-size: 13px;
}

.empty-list {
  padding: 24px 14px;
  text-align: center;
}

.empty-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.empty-detail h3 {
  color: var(--text-dim);
  font-size: 15px;
  font-weight: 600;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-muted);
}

.status-dot.status-running,
.status-dot.status-starting {
  background: var(--green);
}

.status-dot.status-stopping {
  background: var(--orange);
}

.status-dot.status-failed {
  background: var(--red);
}

.status-dot.status-exited,
.status-dot.status-stopped {
  background: var(--text-muted);
}

.badge {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 999px;
}

.badge.status-running,
.badge.status-starting {
  background: var(--green-dim);
  color: var(--green);
}

.badge.status-stopping {
  background: var(--orange-dim);
  color: var(--orange);
}

.badge.status-failed {
  background: var(--red-dim);
  color: var(--red);
}

.badge.status-exited,
.badge.status-stopped {
  background: var(--surface2);
  color: var(--text-muted);
}

.detail-title .badge {
  white-space: nowrap;
}

.btn-primary {
  background: var(--accent);
  color: #fff;
  border: none;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
}

.btn-primary:hover {
  background: var(--accent-hover);
}

.btn-ghost,
.btn-sm {
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.btn-ghost {
  padding: 6px 12px;
}

.btn-sm {
  padding: 4px 10px;
}

.btn-ghost:hover,
.btn-sm:hover {
  background: var(--surface2);
  color: var(--text);
}

.btn-sm:disabled {
  cursor: not-allowed;
  opacity: .45;
}

.btn-green {
  border-color: var(--green);
  color: var(--green);
}

.btn-green:hover {
  background: var(--green-dim);
}

.btn-red {
  border-color: var(--red);
  color: var(--red);
}

.btn-red:hover {
  background: var(--red-dim);
}

.btn-xs {
  padding: 2px 8px;
  font-size: 11px;
}

@media (max-width: 860px) {
  .services-shell {
    grid-template-columns: 1fr;
  }

  .service-list-pane {
    max-height: 220px;
    border-right: none;
    border-bottom: 1px solid var(--border);
  }

  .detail-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .meta-grid {
    grid-template-columns: 1fr;
  }
}
</style>
