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

const isNew = computed(() => store.selectedScriptID === -1)
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
.script-config { padding: 8px; }
.config-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.config-header h2 { font-size: 15px; font-weight: 600; color: var(--text); }
.header-actions { display: flex; gap: 6px; align-items: center; }
.badge-running { background: #1b5e20; color: #4caf50; padding: 3px 10px; border-radius: 12px; font-size: 12px; }
.btn-run  { background: #2e7d32; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-stop { background: #c62828; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-save { background: var(--accent); color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-copy { background: #6a1b9a; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-timer { background: #e65100; color: #fff; border: none; padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.btn-delete { background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); padding: 5px 12px; border-radius: 4px; font-size: 13px; }
.form-body { border: none; }
.form-row { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.form-row > label:first-child { width: 76px; font-size: 12px; color: var(--text-muted); flex-shrink: 0; }
.label-inline { font-size: 12px; color: var(--text-muted); flex-shrink: 0; white-space: nowrap; }
.form-row input, .form-row select { flex: 1; padding: 5px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: 13px; }
.form-row input:focus, .form-row select:focus { border-color: var(--accent); outline: none; }
.form-row > button { padding: 4px 10px; background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; flex-shrink: 0; }
.mode-toggle { display: flex; border: 1px solid var(--border); border-radius: 4px; overflow: hidden; }
.mode-btn { padding: 4px 16px; background: var(--input-bg); color: var(--text-muted); border: none; font-size: 12px; cursor: pointer; transition: background .15s; }
.mode-btn.active { background: var(--accent); color: #fff; }
.form-section { margin-top: 16px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.section-header span { font-size: 12px; color: var(--text-muted); }
.btn-add-env { padding: 3px 10px; background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; }
.env-row { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.env-row input { flex: 1; padding: 4px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: 12px; }
.env-eq { color: var(--text-muted); font-size: 13px; flex-shrink: 0; }
.btn-rm { padding: 3px 7px; background: none; color: var(--text-muted); border: none; font-size: 13px; }
.btn-rm:hover { color: #e74c3c; }
fieldset:disabled { opacity: 0.5; pointer-events: none; }
.toast { position: fixed; bottom: 320px; left: 50%; transform: translateX(-50%); background: #2e7d32; color: #fff; padding: 7px 20px; border-radius: 20px; font-size: 13px; z-index: 200; pointer-events: none; }
</style>
