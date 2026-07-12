<template>
  <div class="services-view">
    <div class="sv-header ui-page-header">
      <div>
        <h2>服务管理</h2>
        <span class="sv-subtitle">{{ services.length }} 个服务</span>
      </div>
      <button class="ui-btn primary" @click="openAdd"><UiIcon name="add" />新增服务</button>
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
              <button class="directory-action-btn" type="button" :disabled="!form.workDir" title="在 VS Code 中打开工作目录" aria-label="在 VS Code 中打开工作目录" @click="openWorkDir(form.workDir, 'vscode', true)"><UiIcon name="terminal" :size="15" /></button>
              <button class="directory-action-btn" type="button" :disabled="!form.workDir" title="在文件夹中打开工作目录" aria-label="在文件夹中打开工作目录" @click="openWorkDir(form.workDir, 'explorer', true)"><UiIcon name="folderOpen" :size="15" /></button>
            </div>
          </label>

          <div class="endpoint-fields">
            <label class="field">
              <span>访问协议</span>
              <select v-model="form.protocol" class="inp">
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
              </select>
            </label>
            <label class="field">
              <span>访问端口（可选）</span>
              <input v-model="form.port" class="inp" type="number" min="1" max="65535" placeholder="8000" />
            </label>
          </div>

          <label class="toggle-row">
            <input type="checkbox" v-model="form.autoStart" />
            <span>PyLot 启动时自动运行</span>
          </label>

          <div v-if="formError" class="form-error">{{ formError }}</div>
          <div class="form-actions">
            <button class="btn-primary" :disabled="formSaving" @click="submitForm">{{ formSaving ? '保存中...' : (editId ? '保存' : '创建') }}</button>
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
              <button class="btn-sm btn-red" @click="showDeleteId = selected.id">删除</button>
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
              <div class="workdir-value">
                <code>{{ selected.work_dir || '未设置' }}</code>
                <span class="workdir-actions">
                  <button class="icon-btn-xs" :disabled="!selected.work_dir" title="在 VS Code 中打开工作目录" aria-label="在 VS Code 中打开工作目录" @click="openWorkDir(selected.work_dir, 'vscode')"><UiIcon name="terminal" :size="13" /></button>
                  <button class="icon-btn-xs" :disabled="!selected.work_dir" title="在文件夹中打开工作目录" aria-label="在文件夹中打开工作目录" @click="openWorkDir(selected.work_dir, 'explorer')"><UiIcon name="folderOpen" :size="13" /></button>
                </span>
              </div>
            </div>
            <div v-if="selected.port">
              <span>访问地址</span>
              <div class="address-row">
                <code>{{ selected.url }}</code>
                <button class="btn-ghost btn-xs icon-text-btn" @click="openServiceURL(selected.url)"><UiIcon name="externalLink" :size="13" />打开</button>
              </div>
            </div>
            <div v-if="selected.port">
              <span>端口状态</span>
              <div class="port-status-row">
                <span class="port-status-info">
                  <strong :class="{ 'port-listening': selectedPortStatus?.listening }">{{ portStatusLabel }}</strong>
                  <small v-if="listenerBelongsToSelected">当前服务的子进程</small>
                </span>
                <button class="icon-btn-xs" title="刷新端口状态" @click="loadPortStatus(selected.id)"><UiIcon name="refresh" :size="13" /></button>
              </div>
            </div>
            <div>
              <span>启动进程 PID</span>
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
  <div v-if="portConflict" class="modal-overlay" @click.self="closePortConflict">
    <div class="conflict-dialog" role="dialog" aria-modal="true" aria-labelledby="port-conflict-title">
      <div class="conflict-header">
        <div>
          <h3 id="port-conflict-title">端口 {{ portConflict.port }} 已被占用</h3>
          <span>结束占用进程后将立即启动服务</span>
        </div>
        <button class="icon-btn-xs" title="取消" :disabled="resolvingConflict" @click="closePortConflict"><UiIcon name="x" :size="14" /></button>
      </div>
      <div class="conflict-details">
        <div><span>PID</span><strong>{{ portConflict.pid }}</strong></div>
        <div><span>进程</span><strong>{{ portConflict.process_name || '未知进程' }}</strong></div>
        <div v-if="portConflict.managed_service_name"><span>PyLot 服务</span><strong>{{ portConflict.managed_service_name }}</strong></div>
        <div v-if="portConflict.process_path" class="conflict-path"><span>路径</span><code>{{ portConflict.process_path }}</code></div>
      </div>
      <div v-if="conflictError" class="form-error">{{ conflictError }}</div>
      <div class="form-actions conflict-actions">
        <button class="btn-ghost" :disabled="resolvingConflict" @click="closePortConflict">取消</button>
        <button class="btn-sm btn-red" :disabled="resolvingConflict" @click="resolvePortConflict">{{ resolvingConflict ? '正在处理...' : '结束进程并启动' }}</button>
      </div>
    </div>
  </div>
  <ConfirmDialog v-if="showDeleteId !== null" title="删除服务？" :message="`服务“${selected?.name || ''}”的配置将被删除，此操作无法撤销。`" @confirm="del(showDeleteId)" @cancel="showDeleteId = null" />
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { BrowserOpenURL, EventsOff, EventsOn } from '../../wailsjs/runtime/runtime.js'
import {
  AddService,
  ClearServiceLogs,
  DeleteService,
  GetServiceLogs,
  GetServicePortStatus,
  ListServices,
  OpenDirectoryDialog,
  OpenInFileExplorer,
  OpenInVSCode,
  RestartService,
  SetServiceAutoStart,
  StartService,
  StopService,
  TerminatePortOwnerAndStartService,
  UpdateService,
} from '../../wailsjs/go/main/App.js'
import ConfirmDialog from './ConfirmDialog.vue'
import UiIcon from './UiIcon.vue'
import { useMainStore } from '../stores/main.js'

const store = useMainStore()
const services = ref([])
const selectedId = ref(null)
const showForm = ref(false)
const editId = ref(null)
const form = reactive({ name: '', command: '', workDir: '', autoStart: false, port: '', protocol: 'http' })
const formError = ref('')
const actionError = ref('')
const logs = reactive({})
const logEl = ref(null)
const showDeleteId = ref(null)
const formSaving = ref(false)
let formHydrating = false
const portStatuses = reactive({})
const portConflict = ref(null)
const conflictServiceId = ref(null)
const resolvingConflict = ref(false)
const conflictError = ref('')
const portProbeTimers = new Map()

const selected = computed(() => services.value.find(s => s.id === selectedId.value) || null)
const currentLogs = computed(() => selectedId.value === null ? [] : (logs[selectedId.value] || []))
const selectedPortStatus = computed(() => selectedId.value === null ? null : (portStatuses[selectedId.value] || null))
const portStatusLabel = computed(() => {
  const status = selectedPortStatus.value
  if (!status) return '检测中'
  if (status.listening) return `正在监听 · ${status.process_name || '未知进程'} · PID ${status.pid}`
  if (selected.value?.running) return '进程运行中，端口尚未监听'
  return '未监听'
})
const listenerBelongsToSelected = computed(() => {
  const status = selectedPortStatus.value
  return status?.listening && status.managed_service_id === selectedId.value && status.pid !== selected.value?.pid
})
watch(form, () => {
  if (showForm.value && !formHydrating) store.markDirty()
})

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
  if (selectedId.value !== null) await loadPortStatus(selectedId.value)
}

async function loadLogs(id) {
  logs[id] = await GetServiceLogs(id) || []
  scrollLog()
}

function openAdd() {
  formHydrating = true
  editId.value = null
  Object.assign(form, { name: '', command: '', workDir: '', autoStart: false, port: '', protocol: 'http' })
  formError.value = ''
  actionError.value = ''
  showForm.value = true
  nextTick(() => { formHydrating = false })
}

function openEdit(s) {
  formHydrating = true
  editId.value = s.id
  Object.assign(form, {
    name: s.name,
    command: s.command,
    workDir: s.work_dir,
    autoStart: s.auto_start,
    port: s.port || '',
    protocol: s.protocol || 'http',
  })
  formError.value = ''
  actionError.value = ''
  showForm.value = true
  nextTick(() => { formHydrating = false })
}

function closeForm() {
  showForm.value = false
  editId.value = null
  formError.value = ''
  store.clearDirty()
}

async function chooseWorkDir() {
  const dir = await OpenDirectoryDialog('选择服务工作目录')
  if (dir) form.workDir = dir
}

async function submitForm() {
  if (formSaving.value) return
  if (!form.name || !form.command) {
    formError.value = '名称和命令不能为空'
    return
  }

  const port = form.port === '' || form.port === null ? 0 : Number(form.port)
  if (!Number.isInteger(port) || port < 0 || port > 65535) {
    formError.value = '端口必须在 1-65535 之间，留空表示不管理端口'
    return
  }

  formSaving.value = true
  try {
    const targetName = form.name
    const targetId = editId.value
    if (targetId) {
      await UpdateService(targetId, form.name, form.command, form.workDir, form.autoStart, port, form.protocol)
    } else {
      await AddService(form.name, form.command, form.workDir, form.autoStart, port, form.protocol)
    }
    closeForm()
    store.clearDirty()
    await load()
    if (targetId) selectedId.value = targetId
    else selectedId.value = services.value.find(s => s.name === targetName)?.id ?? selectedId.value
    if (selectedId.value !== null) await loadLogs(selectedId.value)
  } catch (e) {
    formError.value = normalizeError(e)
  } finally {
    formSaving.value = false
  }
}

async function start(s) {
  actionError.value = ''
  selectedId.value = s.id
  ensureLogs(s.id)
  Object.assign(s, { status: 'starting', running: false })
  try {
    await StartService(s.id)
  } catch (e) {
    const handled = await handleActionFailure(s.id, e, '启动失败')
    Object.assign(s, { status: handled ? 'stopped' : 'failed', running: false })
  }
}

async function stop(id) {
  actionError.value = ''
  const service = services.value.find(item => item.id === id)
  if (service) service.status = 'stopping'
  try {
    await StopService(id)
  } catch (e) {
    actionError.value = normalizeError(e)
    if (service) service.status = service.running ? 'running' : 'failed'
  }
}

async function restart(id) {
  actionError.value = ''
  const service = services.value.find(item => item.id === id)
  if (service) service.status = 'stopping'
  try {
    await RestartService(id)
  } catch (e) {
    await handleActionFailure(id, e, '重启失败')
  }
}

async function del(id) {
  showDeleteId.value = null
  actionError.value = ''
  try {
    await DeleteService(id)
    delete logs[id]
    services.value = services.value.filter(service => service.id !== id)
    if (selectedId.value === id) selectedId.value = services.value[0]?.id ?? null
  } catch (e) {
    actionError.value = normalizeError(e)
  }
}

async function select(s) {
  if (showForm.value && store.isDirty) {
    store.requestNavigation(() => selectService(s))
    return
  }
  await selectService(s)
}

async function selectService(s) {
  selectedId.value = s.id
  showForm.value = false
  store.clearDirty()
  actionError.value = ''
  await loadLogs(s.id)
  await loadPortStatus(s.id)
}

async function openWorkDir(dir, target, inForm = false) {
  if (!dir) return
  if (inForm) formError.value = ''
  else actionError.value = ''
  try {
    if (target === 'vscode') await OpenInVSCode(dir)
    else await OpenInFileExplorer(dir)
  } catch (error) {
    if (inForm) formError.value = normalizeError(error)
    else actionError.value = normalizeError(error)
  }
}

async function loadPortStatus(id) {
  const service = services.value.find(item => item.id === id)
  if (!service?.port) {
    delete portStatuses[id]
    return null
  }
  try {
    const status = await GetServicePortStatus(id)
    portStatuses[id] = status
    return status
  } catch (e) {
    actionError.value = normalizeError(e)
    return null
  }
}

function openServiceURL(url) {
  if (url) BrowserOpenURL(url)
}

async function handleActionFailure(id, error, prefix) {
  const message = normalizeError(error)
  appendLocalLog(id, `${prefix}: ${message}`, true)
  const isPortConflict = message.includes('端口') && (message.includes('占用') || message.includes('未在'))
  if (isPortConflict) {
    const status = await loadPortStatus(id)
    if (status?.listening) {
      conflictServiceId.value = id
      portConflict.value = status
      conflictError.value = ''
      return true
    }
  }
  actionError.value = message
  return false
}

function probePortUntilListening(id, attempts = 10) {
  const previous = portProbeTimers.get(id)
  if (previous) clearTimeout(previous)
  const timer = setTimeout(async () => {
    portProbeTimers.delete(id)
    const status = await loadPortStatus(id)
    const service = services.value.find(item => item.id === id)
    if (!status?.listening && service?.running && attempts > 1) probePortUntilListening(id, attempts - 1)
  }, 500)
  portProbeTimers.set(id, timer)
}

function closePortConflict() {
  if (resolvingConflict.value) return
  portConflict.value = null
  conflictServiceId.value = null
  conflictError.value = ''
}

async function resolvePortConflict() {
  if (!portConflict.value || conflictServiceId.value === null) return
  resolvingConflict.value = true
  conflictError.value = ''
  const id = conflictServiceId.value
  try {
    await TerminatePortOwnerAndStartService(id, portConflict.value.pid)
    resolvingConflict.value = false
    closePortConflict()
    await load()
  } catch (e) {
    conflictError.value = normalizeError(e)
    const status = await loadPortStatus(id)
    if (status?.listening) portConflict.value = status
  } finally {
    resolvingConflict.value = false
  }
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
  if (d.status === 'starting') logs[d.id] = []
  Object.assign(s, {
    running: d.running,
    status: d.status,
    pid: d.pid,
    started_at: d.started_at,
    stopped_at: d.stopped_at,
    exit_code: d.exit_code,
    last_error: d.last_error,
  })
  if (s.port) {
    if (d.running) probePortUntilListening(s.id)
    else setTimeout(() => loadPortStatus(s.id), 100)
  }
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
  for (const timer of portProbeTimers.values()) clearTimeout(timer)
  portProbeTimers.clear()
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
.input-action,
.address-row,
.port-status-row,
.icon-text-btn {
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

.endpoint-fields {
  display: grid;
  grid-template-columns: minmax(120px, .4fr) minmax(180px, 1fr);
  gap: 12px;
}

.input-action .inp {
  flex: 1;
}

.directory-action-btn {
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
}

.directory-action-btn:hover:not(:disabled) {
  background: var(--surface2);
  color: var(--text);
}

.directory-action-btn:disabled,
.icon-btn-xs:disabled {
  opacity: .4;
}

.workdir-value,
.workdir-actions {
  min-width: 0;
  display: flex;
  align-items: center;
}

.workdir-value {
  justify-content: space-between;
  gap: 8px;
}

.workdir-value code {
  min-width: 0;
}

.workdir-actions {
  flex: 0 0 auto;
  gap: 4px;
}

.address-row,
.port-status-row {
  min-width: 0;
  justify-content: space-between;
  gap: 8px;
}

.address-row code {
  min-width: 0;
}

.icon-text-btn {
  gap: 4px;
}

.icon-btn-xs {
  width: 26px;
  height: 26px;
  flex: 0 0 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
}

.icon-btn-xs:hover {
  background: var(--surface2);
  color: var(--text);
}

.port-listening {
  color: var(--green) !important;
}

.port-status-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.port-status-info strong,
.port-status-info small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.port-status-info small {
  color: var(--text-muted);
  font-size: 11px;
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

.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, .6);
}

.conflict-dialog {
  width: min(480px, 100%);
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg);
  box-shadow: 0 16px 48px rgba(0, 0, 0, .3);
}

.conflict-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.conflict-header h3 {
  margin-bottom: 4px;
  font-size: 15px;
  font-weight: 600;
}

.conflict-header span,
.conflict-details span {
  color: var(--text-muted);
  font-size: 12px;
}

.conflict-details {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
}

.conflict-details > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.conflict-details strong,
.conflict-details code {
  overflow: hidden;
  color: var(--text-dim);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conflict-path {
  grid-column: 1 / -1;
}

.conflict-actions {
  justify-content: flex-end;
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

  .endpoint-fields,
  .conflict-details {
    grid-template-columns: 1fr;
  }

  .conflict-path {
    grid-column: auto;
  }
}
</style>
