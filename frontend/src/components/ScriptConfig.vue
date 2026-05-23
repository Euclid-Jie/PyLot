<template>
  <div class="script-config">
    <div class="config-header">
      <h2>{{ isNew ? '新建脚本' : form.name || '脚本配置' }}</h2>
      <div class="header-actions">
        <span v-if="isRunning" class="badge-running">● 运行中</span>
        <button v-if="!isRunning" class="btn-run" @click="handleRun">▶ 运行</button>
        <button v-if="isRunning" class="btn-stop" @click="handleStop">■ 停止</button>
        <button class="btn-save" @click="handleSave">保存</button>
        <button v-if="!isNew" class="btn-copy" @click="handleCopy">复制</button>
        <button class="btn-timer" @click="showTimer = true">⏰ 定时</button>
        <button v-if="!isNew" class="btn-delete" @click="handleDelete">删除</button>
      </div>
    </div>

    <fieldset :disabled="isRunning" class="form-body">
      <!-- 名称 + 分类 同行 -->
      <div class="form-row">
        <label>名称</label>
        <input v-model="form.name" style="flex:2" />
        <label class="label-inline">分类</label>
        <select v-model="form.category" style="flex:1">
          <option value="crawler">爬取上传</option>
          <option value="processor">数据处理</option>
          <option value="tool">个人工具</option>
        </select>
      </div>

      <!-- 启动模式：大按钮切换 -->
      <div class="form-row">
        <label>启动模式</label>
        <div class="mode-toggle">
          <button :class="['mode-btn', { active: form.launchMode === 'script' }]" @click="form.launchMode = 'script'">script</button>
          <button :class="['mode-btn', { active: form.launchMode === 'module' }]" @click="form.launchMode = 'module'">module</button>
        </div>
      </div>

      <div class="form-row">
        <label>解释器</label>
        <input v-model="form.interpreterPath" />
        <button @click="browse('interpreter')">浏览</button>
      </div>
      <div class="form-row">
        <label>脚本路径</label>
        <input v-model="form.scriptPath" />
        <button @click="browse('script')">浏览</button>
      </div>
      <div class="form-row">
        <label>工作目录</label>
        <input v-model="form.workDir" />
        <button @click="browseDir">浏览</button>
      </div>

      <!-- 固定参数 + 超时 同行 -->
      <div class="form-row">
        <label>固定参数</label>
        <input v-model="form.fixedArgs" placeholder="--env prod --debug" style="flex:3" />
        <label class="label-inline">超时(s)</label>
        <input type="number" v-model.number="form.timeoutSeconds" min="0" placeholder="0=∞" style="width:72px;flex:none" />
      </div>

      <div class="form-section">
        <div class="section-header">
          <span>私有环境变量</span>
          <button class="btn-add-env" @click="envPairs.push({ key: '', val: '' })">+ 添加</button>
        </div>
        <div v-for="(kv, i) in envPairs" :key="i" class="env-row">
          <input v-model="kv.key" placeholder="KEY" />
          <span class="env-eq">=</span>
          <input v-model="kv.val" placeholder="VALUE" />
          <button class="btn-rm" @click="envPairs.splice(i, 1)">✕</button>
        </div>
      </div>
    </fieldset>
  </div>

  <TempArgsModal v-if="showTempArgs" :fixedArgs="form.fixedArgs" @run="doRun" @close="showTempArgs = false" />
  <TimerModal v-if="showTimer" :scriptId="form.id" @close="showTimer = false" />
  <div v-if="toast" class="toast">{{ toast }}</div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { GetScripts, CreateScript, UpdateScript, DeleteScript, RunScript, StopScript, OpenFileDialog, OpenDirectoryDialog, InferFromScriptPath } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import TempArgsModal from './TempArgsModal.vue'
import TimerModal from './TimerModal.vue'

const store = useMainStore()
const showTempArgs = ref(false)
const showTimer = ref(false)

const form = ref({
  id: null, name: '', category: 'crawler', interpreterPath: 'python',
  workDir: '', scriptPath: '', launchMode: 'script', fixedArgs: '',
  timeoutSeconds: 0, privateEnv: '{}'
})
const envPairs = ref([])

const isNew = computed(() => store.selectedScriptID === 0)
const isRunning = computed(() => store.runningScripts.has(store.selectedScriptID))

onMounted(loadScript)
watch(() => store.selectedScriptID, loadScript)
watch(() => form.value.workDir, v => { store.selectedScriptWorkDir = v || '' })

async function loadScript() {
  if (isNew.value) {
    form.value = { id: null, name: '', category: 'crawler', interpreterPath: 'python', workDir: '', scriptPath: '', launchMode: 'script', fixedArgs: '', timeoutSeconds: 0, privateEnv: '{}' }
    envPairs.value = []
    return
  }
  const scripts = await GetScripts()
  const s = scripts?.find(x => x.id === store.selectedScriptID)
  if (s) {
    form.value = { ...s }
    store.selectedScriptWorkDir = s.workDir || ''
    try { const obj = JSON.parse(s.privateEnv || ''); envPairs.value = Object.entries(obj).map(([k, v]) => ({ key: k, val: v })) }
    catch { envPairs.value = [] }
  }
}

function buildPrivateEnv() {
  const obj = {}
  envPairs.value.forEach(({ key, val }) => { if (key) obj[key] = val })
  return JSON.stringify(obj)
}

async function handleSave() {
  const data = { ...form.value, privateEnv: buildPrivateEnv() }
  if (isNew.value) {
    const id = await CreateScript(data)
    store.setScript(id)
  } else {
    await UpdateScript(data)
  }
  store.refreshScriptList()
  showToast('保存成功')
}

const toast = ref('')
function showToast(msg) {
  toast.value = msg
  setTimeout(() => { toast.value = '' }, 2000)
}

function handleRun() {
  if (form.value.fixedArgs) showTempArgs.value = true
  else doRun('')
}

async function doRun(args) {
  showTempArgs.value = false
  await RunScript(store.selectedScriptID, args)
}

async function handleStop() {
  await StopScript(store.selectedScriptID)
}

async function handleDelete() {
  if (!confirm('确认删除此脚本？')) return
  await DeleteScript(store.selectedScriptID)
  store.refreshScriptList()
  store.setScript(null)
  store.setView('script')
}

async function handleCopy() {
  const data = { ...form.value, id: null, name: form.value.name + ' (副本)', privateEnv: buildPrivateEnv() }
  const id = await CreateScript(data)
  store.setScript(id)
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
</script>

<style scoped>
.script-config { padding: 0; }
.config-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); }
.config-header h2 { font-size: 20px; font-weight: 600; color: var(--text); }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.badge-running { background: var(--green-dim); color: var(--green); padding: 5px 12px; border-radius: 20px; font-size: 13px; font-weight: 500; border: 1px solid rgba(63,185,80,.3); }
.btn-run, .btn-stop, .btn-save, .btn-copy, .btn-timer, .btn-delete { padding: 5px 14px; border-radius: var(--radius); font-size: 14px; font-weight: 500; transition: background .12s, opacity .12s; }
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
.form-row > label:first-child { width: 80px; font-size: 14px; color: var(--text-dim); flex-shrink: 0; text-align: right; }
.label-inline { font-size: 14px; color: var(--text-dim); flex-shrink: 0; white-space: nowrap; }
.form-row input, .form-row select { flex: 1; padding: 7px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: 14px; transition: border-color .12s, box-shadow .12s; }
.form-row input:focus, .form-row select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.form-row > button { padding: 6px 12px; background: var(--surface2); color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; flex-shrink: 0; transition: background .12s, color .12s; }
.form-row > button:hover { background: var(--surface); color: var(--text); border-color: var(--text-muted); }
.mode-toggle { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.mode-btn { padding: 6px 20px; background: transparent; color: var(--text-muted); border: none; font-size: 14px; font-weight: 500; cursor: pointer; transition: background .12s, color .12s; }
.mode-btn.active { background: var(--accent); color: #fff; }
.mode-btn:not(.active):hover { background: var(--surface2); color: var(--text); }
.form-section { margin-top: 20px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; padding-bottom: 8px; border-bottom: 1px solid var(--border); }
.section-header span { font-size: 12px; font-weight: 600; letter-spacing: .04em; text-transform: uppercase; color: var(--text-muted); }
.btn-add-env { padding: 4px 10px; background: transparent; color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; transition: background .12s, color .12s; }
.btn-add-env:hover { background: var(--surface2); color: var(--text); }
.env-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.env-row input { flex: 1; padding: 6px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: 14px; }
.env-row input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.env-eq { color: var(--text-muted); font-size: 14px; flex-shrink: 0; }
.btn-rm { padding: 4px 8px; background: none; color: var(--text-muted); border: none; font-size: 14px; transition: color .12s; }
.btn-rm:hover { color: var(--red); }
fieldset:disabled { opacity: 0.4; pointer-events: none; }
.toast { position: fixed; bottom: 280px; left: 50%; transform: translateX(-50%); background: var(--green); color: #fff; padding: 8px 20px; border-radius: 20px; font-size: 14px; font-weight: 500; z-index: 200; pointer-events: none; box-shadow: 0 4px 12px rgba(0,0,0,.3); }
</style>
