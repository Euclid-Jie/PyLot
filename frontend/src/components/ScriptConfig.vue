<template>
  <div class="script-config" @keydown.ctrl.s.prevent="handleSave">
    <div class="config-header ui-page-header">
      <div class="config-title">
        <button class="mobile-back ui-icon-btn" title="返回脚本列表" aria-label="返回脚本列表" @click="store.closeScriptDetail()"><UiIcon name="chevronLeft" /></button>
        <div>
          <h2>{{ isNew ? '新建脚本' : form.name || '脚本配置' }}</h2>
          <p class="ui-page-subtitle">配置运行环境、参数和私有环境变量</p>
        </div>
      </div>
      <div class="header-actions">
        <span v-if="isRunning" class="ui-status running">运行中</span>
        <button v-if="!isRunning" class="ui-btn primary" :disabled="isNew || store.isDirty || !!actionPending" :title="isNew || store.isDirty ? '请先保存脚本' : '运行脚本'" @click="handleRun"><UiIcon name="play" />{{ actionPending === 'run' ? '启动中' : '运行' }}</button>
        <button v-else class="ui-btn danger" :disabled="!!actionPending" @click="handleStop"><UiIcon name="stop" />{{ actionPending === 'stop' ? '停止中' : '停止' }}</button>
        <button class="ui-btn" :disabled="saving || isRunning" @click="handleSave"><UiIcon name="save" />{{ saving ? '保存中' : '保存' }}</button>
        <button v-if="!isNew" class="ui-icon-btn" :disabled="isRunning || !!actionPending" title="复制脚本" aria-label="复制脚本" @click="handleCopy"><UiIcon name="copy" /></button>
        <button class="ui-icon-btn" :disabled="isNew || isRunning" :title="isNew ? '请先保存脚本' : '设置定时任务'" aria-label="设置定时任务" @click="showTimer = true"><UiIcon name="clock" /></button>
        <button v-if="!isNew" class="ui-icon-btn danger-icon" :disabled="isRunning || !!actionPending" title="删除脚本" aria-label="删除脚本" @click="showDeleteConfirm = true"><UiIcon name="delete" /></button>
      </div>
    </div>

    <div v-if="actionError" class="action-message">{{ actionError }}</div>
    <fieldset :disabled="isRunning" class="form-body">
      <div class="form-row">
        <label>名称</label>
        <input v-model="form.name" class="ui-input" style="flex:2" />
        <label class="label-inline">列表</label>
        <select v-model.number="form.listId" class="ui-input" style="flex:1">
          <option :value="0">未分类</option>
          <option v-for="list in scriptLists" :key="list.id" :value="list.id">{{ list.name }}</option>
        </select>
      </div>

      <div class="form-row">
        <label>启动模式</label>
        <div class="mode-toggle">
          <button :class="['mode-btn', { active: form.launchMode === 'script' }]" @click="form.launchMode = 'script'">script</button>
          <button :class="['mode-btn', { active: form.launchMode === 'module' }]" @click="form.launchMode = 'module'">module</button>
          <button :class="['mode-btn', { active: form.launchMode === 'custom' }]" @click="form.launchMode = 'custom'">custom</button>
        </div>
        <span style="flex:1"></span>
        <label class="label-inline">超时(s)</label>
        <input type="number" v-model.number="form.timeoutSeconds" class="ui-input" min="0" placeholder="0=∞" style="width:80px;flex:none" />
      </div>

      <div v-if="form.launchMode !== 'custom'" class="form-row">
        <label>解释器</label>
        <input v-model="form.interpreterPath" class="ui-input ui-mono" /><button class="ui-btn" @click="browse('interpreter')"><UiIcon name="folder" />选择</button>
      </div>
      <div v-if="form.launchMode !== 'custom'" class="form-row">
        <label>脚本路径</label>
        <input v-model="form.scriptPath" class="ui-input ui-mono" />
        <button class="ui-btn" @click="browse('script')"><UiIcon name="folder" />选择</button>
      </div>
      <div v-else class="form-row">
        <label>命令</label>
        <input v-model="form.scriptPath" class="ui-input ui-mono" placeholder='.venv\Scripts\python.exe -m fof99_nav_ingestion run --include-stale --update-token' />
      </div>
      <div class="form-row">
        <label>工作目录</label>
        <input v-model="form.workDir" class="ui-input ui-mono" />
        <button class="ui-btn" @click="browseDir"><UiIcon name="folder" />选择</button>
        <button class="ui-icon-btn workdir-icon-btn" :disabled="!form.workDir" title="在 VS Code 中打开工作目录" aria-label="在 VS Code 中打开工作目录" @click="openWorkDir"><UiIcon name="terminal" /></button>
      </div>

      <div v-if="form.launchMode !== 'custom'" class="form-section">
        <div class="section-header">
          <span>固定参数</span>
          <button class="ui-btn small" @click="argPairs.push({ flag: '', val: '' })"><UiIcon name="add" />添加参数</button>
        </div>
        <div v-for="(arg, i) in argPairs" :key="'arg'+i" class="env-row">
          <span class="arg-prefix">--</span>
          <input v-model="arg.flag" placeholder="begin" style="flex:1" />
          <span class="env-eq"> </span>
          <input v-model="arg.val" placeholder="value" style="flex:2" />
          <button class="ui-icon-btn danger-icon" title="移除参数" aria-label="移除参数" @click="argPairs.splice(i, 1)"><UiIcon name="x" /></button>
        </div>
      </div>

      <div class="form-section">
        <div class="section-header">
          <span>私有环境变量</span>
          <button class="ui-btn small" @click="envPairs.push({ key: '', val: '' })"><UiIcon name="add" />添加变量</button>
        </div>
        <div v-for="(kv, i) in envPairs" :key="i" class="env-row">
          <input v-model="kv.key" placeholder="KEY" />
          <span class="env-eq">=</span>
          <input v-model="kv.val" placeholder="VALUE" />
          <button class="ui-icon-btn danger-icon" title="移除变量" aria-label="移除变量" @click="envPairs.splice(i, 1)"><UiIcon name="x" /></button>
        </div>
      </div>
    </fieldset>
  </div>

  <TempArgsModal v-if="showTempArgs" :fixedArgs="buildFixedArgs()" @run="doRun" @close="showTempArgs = false" />
  <TimerModal v-if="showTimer" :scriptId="form.id" @close="showTimer = false" />
  <ConfirmDialog v-if="showDeleteConfirm" title="删除脚本？" :message="`脚本“${form.name}”及其配置将被删除，此操作无法撤销。`" @confirm="handleDelete" @cancel="showDeleteConfirm = false" />
  <div v-if="toast" class="ui-toast">{{ toast }}</div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { GetScript, GetScriptLists, CreateScript, UpdateScript, DeleteScript, RunScript, StopScript, OpenFileDialog, OpenDirectoryDialog, OpenInVSCode, InferFromScriptPath } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import TempArgsModal from './TempArgsModal.vue'
import TimerModal from './TimerModal.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const showTempArgs = ref(false)
const showTimer = ref(false)
const showDeleteConfirm = ref(false)
const saving = ref(false)
const actionPending = ref('')
const actionError = ref('')
const scriptLists = ref([])
let hydrating = true

const form = ref({
  id: null, name: '', category: '', listId: 0, interpreterPath: 'python',
  workDir: '', scriptPath: '', launchMode: 'script', fixedArgs: '',
  timeoutSeconds: 0, privateEnv: '{}'
})
const envPairs = ref([])
const argPairs = ref([])  // { flag: string, val: string }

const isNew = computed(() => store.selectedScriptID === 0)
const isRunning = computed(() => store.runningScripts.has(store.selectedScriptID))

onMounted(async () => {
  scriptLists.value = await GetScriptLists() || []
  await loadScript()
})
watch(() => store.selectedScriptID, loadScript)
watch(() => store.scriptListVersion, async () => { scriptLists.value = await GetScriptLists() || [] })
watch(() => form.value.workDir, v => { store.selectedScriptWorkDir = v || '' })
watch([form, envPairs, argPairs], () => {
  if (!hydrating) store.markDirty()
}, { deep: true })

async function loadScript() {
  hydrating = true
  actionError.value = ''
  store.clearDirty()
  if (isNew.value) {
    const selectedListId = typeof store.selectedScriptListId === 'number' ? store.selectedScriptListId : 0
    form.value = { id: null, name: '', category: '', listId: selectedListId, interpreterPath: 'python', workDir: '', scriptPath: '', launchMode: 'script', fixedArgs: '', timeoutSeconds: 0, privateEnv: '{}' }
    envPairs.value = []
    argPairs.value = []
    await finishHydration()
    return
  }
  const s = await GetScript(store.selectedScriptID).catch(error => {
    actionError.value = normalizeError(error)
    return null
  })
  if (s) {
    form.value = { ...s }
    store.selectedScriptWorkDir = s.workDir || ''
    // parse "-- flag val --flag2 val2" into [{flag, val}]
    if (s.fixedArgs) {
      const tokens = s.fixedArgs.match(/--(\S+)\s+([^-]\S*)/g) || []
      argPairs.value = tokens.map(t => {
        const m = t.match(/--(\S+)\s+(.+)/)
        return m ? { flag: m[1], val: m[2] } : { flag: t.replace(/^--/, ''), val: '' }
      })
    } else {
      argPairs.value = []
    }
    try { const obj = JSON.parse(s.privateEnv || ''); envPairs.value = Object.entries(obj).map(([k, v]) => ({ key: k, val: v })) }
    catch { envPairs.value = [] }
  }
  await finishHydration()
}

async function finishHydration() {
  await nextTick()
  hydrating = false
}

function buildFixedArgs() {
  return argPairs.value.filter(a => a.flag).map(a => a.val ? `--${a.flag} ${a.val}` : `--${a.flag}`).join(' ')
}

function buildPrivateEnv() {
  const obj = {}
  envPairs.value.forEach(({ key, val }) => { if (key) obj[key] = val })
  return JSON.stringify(obj)
}

async function handleSave() {
  if (saving.value) return
  actionError.value = ''
  if (!form.value.name.trim()) { actionError.value = '请输入脚本名称'; return }
  if (form.value.launchMode === 'custom') {
    if (!form.value.workDir.trim()) { actionError.value = '请选择工作目录'; return }
    if (!form.value.scriptPath.trim()) { actionError.value = '请输入自定义命令'; return }
  } else if (!form.value.scriptPath.trim()) {
    actionError.value = '请选择脚本文件'
    return
  }
  saving.value = true
  const data = { ...form.value, fixedArgs: form.value.launchMode === 'custom' ? '' : buildFixedArgs(), privateEnv: buildPrivateEnv() }
  try {
    if (isNew.value) {
      const id = await CreateScript(data)
      store.clearDirty()
      if (store.selectedScriptListId !== 'all') store.showScriptInList(data.listId)
      store.setScript(id)
    } else {
      await UpdateScript(data)
      store.clearDirty()
      if (store.selectedScriptListId !== 'all') store.showScriptInList(data.listId)
    }
    store.refreshScriptList()
    showToast('保存成功')
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    saving.value = false
  }
}

const toast = ref('')
function showToast(msg) {
  toast.value = msg
  setTimeout(() => { toast.value = '' }, 2000)
}

function handleRun() {
  if (isNew.value || store.isDirty || actionPending.value) return
  if (form.value.launchMode !== 'custom' && argPairs.value.some(a => a.flag)) showTempArgs.value = true
  else doRun('')
}

async function doRun(args) {
  showTempArgs.value = false
  actionPending.value = 'run'
  actionError.value = ''
  store.setRunning(store.selectedScriptID, true)
  try {
    await RunScript(store.selectedScriptID, args)
  } catch (error) {
    store.setRunning(store.selectedScriptID, false)
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function handleStop() {
  if (actionPending.value) return
  actionPending.value = 'stop'
  actionError.value = ''
  try {
    await StopScript(store.selectedScriptID)
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function handleDelete() {
  if (actionPending.value) return
  actionPending.value = 'delete'
  showDeleteConfirm.value = false
  try {
    await DeleteScript(store.selectedScriptID)
    store.clearDirty()
    store.refreshScriptList()
    store.setScript(null)
    store.setView('script')
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function handleCopy() {
  if (actionPending.value) return
  actionPending.value = 'copy'
  actionError.value = ''
  const data = { ...form.value, id: null, name: form.value.name + ' (副本)', fixedArgs: form.value.launchMode === 'custom' ? '' : buildFixedArgs(), privateEnv: buildPrivateEnv() }
  try {
    const id = await CreateScript(data)
    store.clearDirty()
    store.refreshScriptList()
    store.setScript(id)
  } catch (error) {
    actionError.value = normalizeError(error)
  } finally {
    actionPending.value = ''
  }
}

async function browse(type) {
  const p = await OpenFileDialog(type === 'interpreter' ? '选择 Python 解释器' : '选择脚本文件')
  if (!p) return
  if (type === 'interpreter') {
    form.value.interpreterPath = p
  } else {
    form.value.scriptPath = p
    const infer = await InferFromScriptPath(p)
    if (infer.interpreterPath) form.value.interpreterPath = infer.interpreterPath
    if (infer.workDir) form.value.workDir = infer.workDir
  }
}

async function browseDir() {
  const p = await OpenDirectoryDialog('选择工作目录')
  if (p) form.value.workDir = p
}

async function openWorkDir() {
  actionError.value = ''
  try {
    await OpenInVSCode(form.value.workDir)
  } catch (error) {
    actionError.value = normalizeError(error)
  }
}

function normalizeError(error) {
  return String(error?.message || error || '操作失败').replace(/^Error:\s*/i, '')
}
</script>

<style scoped>
.script-config { padding: 0; }
.config-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); }
.config-title { min-width: 0; display: flex; align-items: center; gap: 8px; }
.mobile-back { display: none; }
.config-header h2 { color: var(--text); font-size: var(--type-page-title); font-weight: var(--weight-semibold); }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.badge-running { background: var(--green-dim); color: var(--green); padding: 5px 12px; border-radius: 20px; font-size: var(--type-body); font-weight: var(--weight-medium); border: 1px solid rgba(63,185,80,.3); }
.btn-run, .btn-stop, .btn-save, .btn-copy, .btn-timer, .btn-delete { padding: 5px 14px; border-radius: var(--radius); font-size: var(--type-section-title); font-weight: var(--weight-medium); transition: background .12s, opacity .12s; }
.btn-run   { background: var(--green-dim);  color: var(--green);  border: 1px solid rgba(63,185,80,.4); }
.btn-stop  { background: var(--red-dim);    color: var(--red);    border: 1px solid rgba(248,81,73,.4); }
.btn-save  { background: var(--accent);     color: #fff;          border: 1px solid var(--accent); }
.btn-copy  { background: var(--surface2);   color: var(--text-dim); border: 1px solid var(--border); }
.btn-timer { background: var(--orange-dim); color: var(--orange); border: 1px solid rgba(210,153,34,.4); }
.btn-delete{ background: transparent;       color: var(--text-muted); border: 1px solid var(--border); }
.btn-save:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.btn-run:hover, .btn-stop:hover, .btn-copy:hover, .btn-timer:hover, .btn-delete:hover { opacity: .8; }
.form-body { border: none; }
.form-row { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.form-row > label:first-child { width: 80px; font-size: var(--type-section-title); color: var(--text-dim); flex-shrink: 0; text-align: right; }
.label-inline { font-size: var(--type-section-title); color: var(--text-dim); flex-shrink: 0; white-space: nowrap; }
.form-row input, .form-row select { flex: 1; padding: 7px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: var(--type-section-title); transition: border-color .12s, box-shadow .12s; }
.form-row input:focus, .form-row select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.form-row > button { padding: 6px 12px; background: var(--surface2); color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: var(--type-body); flex-shrink: 0; transition: background .12s, color .12s; }
.form-row > button:hover { background: var(--surface); color: var(--text); border-color: var(--text-muted); }
.mode-toggle { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.mode-btn { padding: 6px 20px; background: transparent; color: var(--text-muted); border: none; font-size: var(--type-section-title); font-weight: var(--weight-medium); cursor: pointer; transition: background .12s, color .12s; }
.mode-btn.active { background: var(--accent); color: #fff; }
.mode-btn:not(.active):hover { background: var(--surface2); color: var(--text); }
.form-section { margin-top: 20px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; padding-bottom: 8px; border-bottom: 1px solid var(--border); }
.section-header span { color: var(--text-muted); font-size: var(--type-section-title); font-weight: var(--weight-semibold); }
.btn-add-env { padding: 4px 10px; background: transparent; color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: var(--type-body); transition: background .12s, color .12s; }
.btn-add-env:hover { background: var(--surface2); color: var(--text); }
.env-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.env-row input { flex: 1; padding: 6px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: var(--type-section-title); }
.env-row input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.env-eq { color: var(--text-muted); font-size: var(--type-section-title); flex-shrink: 0; }
.arg-prefix { color: var(--text-muted); font-family: var(--mono); font-size: var(--type-body); font-weight: var(--weight-regular); flex-shrink: 0; }
.btn-rm { padding: 4px 8px; background: none; color: var(--text-muted); border: none; font-size: var(--type-section-title); transition: color .12s; }
.btn-rm:hover { color: var(--red); }
fieldset:disabled { opacity: 0.4; pointer-events: none; }
.toast { position: fixed; bottom: 280px; left: 50%; transform: translateX(-50%); background: var(--green); color: #fff; padding: 8px 20px; border-radius: 20px; font-size: var(--type-section-title); font-weight: var(--weight-medium); z-index: 200; pointer-events: none; box-shadow: 0 4px 12px rgba(0,0,0,.3); }

/* Unified desktop form treatment. */
.script-config { max-width: 1180px; margin: 0 auto; }
.config-header { margin-bottom: 22px; }
.config-header h2 { font-size: var(--type-page-title); }
.header-actions { gap: 6px; flex-wrap: nowrap; }
.danger-icon:hover { border-color: rgba(248, 81, 73, .35); background: var(--red-dim); color: var(--red); }
.action-message { margin: -10px 0 14px; padding: 8px 10px; border: 1px solid rgba(248, 81, 73, .28); border-radius: var(--radius); background: var(--red-dim); color: var(--red); font-size: var(--type-label); }
.form-body { display: flex; flex-direction: column; gap: 0; }
.form-row { min-height: 44px; margin-bottom: 6px; }
.form-row > label:first-child { width: 88px; color: var(--text-dim); font-size: var(--type-label); font-weight: var(--weight-regular); text-align: left; }
.label-inline { font-size: var(--type-label); font-weight: var(--weight-regular); }
.form-row input, .form-row select, .env-row input { min-height: var(--control-lg); padding: 7px 10px; border-color: var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--text); font-size: var(--type-body); }
.form-row > button.ui-btn { min-height: var(--control-lg); padding: 0 12px; background: var(--surface); color: var(--text-dim); }
.form-row > button.workdir-icon-btn { width: var(--control-lg); height: var(--control-lg); min-height: var(--control-lg); padding: 0; background: var(--surface); }
.mode-toggle { background: var(--input-bg); }
.mode-btn { min-height: 34px; padding: 0 18px; font-size: var(--type-body); }
.mode-btn.active { background: var(--accent-dim); color: var(--accent); }
.form-section { margin-top: 18px; padding: 16px; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--surface); }
.section-header { margin-bottom: 12px; padding-bottom: 10px; }
.section-header span { color: var(--text-dim); font-size: var(--type-section-title); font-weight: var(--weight-semibold); }
.env-row { min-height: 36px; margin-bottom: 7px; }
.env-row:last-child { margin-bottom: 0; }
.env-row .ui-icon-btn { width: 30px; height: 30px; }
fieldset:disabled { opacity: .62; }
@media (max-width: 980px) {
  .config-header { align-items: flex-start; }
  .header-actions { flex-wrap: wrap; justify-content: flex-end; }
  .form-row { flex-wrap: wrap; }
  .form-row > label:first-child { width: 100%; }
}
@media (max-width: 1050px) {
  .mobile-back { display: inline-flex; }
}

</style>
