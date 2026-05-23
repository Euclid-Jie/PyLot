<template>
  <div class="sidebar-inner">
    <div class="sidebar-header">
      <span>脚本管理</span>
      <button class="btn-add" @click="newScript">+</button>
    </div>

    <div v-for="cat in categories" :key="cat.key" class="category">
      <div class="cat-header" @click="cat.open = !cat.open">
        <span>{{ cat.icon }} {{ cat.label }}</span>
        <span>{{ cat.open ? '▾' : '▸' }}</span>
      </div>
      <div v-if="cat.open" class="cat-scripts">
        <div
          v-for="s in scriptsByCategory(cat.key)"
          :key="s.id"
          class="script-item"
          :class="{ active: store.selectedScriptID === s.id }"
          @click="store.setScript(s.id)"
        >
          <span class="script-name">{{ s.name }}</span>
          <span v-if="store.runningScripts.has(s.id)" class="spinner">⟳</span>
        </div>
      </div>
    </div>

    <div class="sidebar-footer">
      <div class="env-section">
        <div class="env-label">全局 .env</div>
        <div class="env-row">
          <input v-model="envPath" placeholder="选择 .env 路径" />
          <button @click="browseEnv">…</button>
        </div>
        <button class="btn-env-save" @click="saveEnv">保存</button>
      </div>
      <button class="btn-schedule" @click="store.setView('schedule')">📅 Schedule</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { GetScripts, GetGlobalConfig, SaveGlobalConfig, OpenFileDialog } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'

const store = useMainStore()
const allScripts = ref([])
const envPath = ref('')

const categories = ref([
  { key: 'crawler', label: '数据爬取上传', icon: '🕷', open: true },
  { key: 'processor', label: '数据处理', icon: '⚙️', open: true },
  { key: 'tool', label: '个人工具', icon: '🔧', open: true },
])

onMounted(async () => {
  await loadScripts()
  const cfg = await GetGlobalConfig()
  envPath.value = cfg.envFilePath || ''
})

watch(() => store.scriptListVersion, loadScripts)

async function loadScripts() {
  allScripts.value = await GetScripts() || []
}

function scriptsByCategory(cat) {
  return allScripts.value.filter(s => s.category === cat)
}

function newScript() {
  store.setScript(-1)
}

async function browseEnv() {
  const p = await OpenFileDialog('选择 .env 文件')
  if (p) envPath.value = p
}

async function saveEnv() {
  await SaveGlobalConfig({ envFilePath: envPath.value })
}
</script>

<style scoped>
.sidebar-inner { display: flex; flex-direction: column; height: 100%; }
.sidebar-header { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; border-bottom: 1px solid #0f3460; font-size: 13px; font-weight: bold; }
.btn-add { background: #533483; color: #fff; border: none; border-radius: 4px; width: 24px; height: 24px; font-size: 16px; line-height: 1; cursor: pointer; }
.category { border-bottom: 1px solid #0f3460; }
.cat-header { display: flex; justify-content: space-between; padding: 8px 12px; font-size: 12px; color: #aaa; cursor: pointer; user-select: none; }
.cat-header:hover { background: #1e2d50; }
.cat-scripts { padding: 2px 0; }
.script-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 16px; font-size: 13px; cursor: pointer; }
.script-item:hover { background: #1e2d50; }
.script-item.active { background: #0f3460; color: #7eb8f7; }
.script-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.spinner { animation: spin 1s linear infinite; display: inline-block; color: #4caf50; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.sidebar-footer { margin-top: auto; padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; border-top: 1px solid #0f3460; }
.env-section { display: flex; flex-direction: column; gap: 4px; }
.env-label { font-size: 11px; color: #888; }
.env-row { display: flex; gap: 4px; }
.env-row input { flex: 1; padding: 3px 6px; background: #1a1a2e; border: 1px solid #444; color: #e0e0e0; border-radius: 3px; font-size: 11px; min-width: 0; }
.env-row button { padding: 3px 6px; background: #3a3a3a; color: #ccc; border: 1px solid #555; border-radius: 3px; font-size: 11px; flex-shrink: 0; }
.btn-env-save { padding: 4px; background: #533483; color: #fff; border: none; border-radius: 3px; font-size: 11px; width: 100%; }
.btn-schedule { width: 100%; padding: 8px; background: #0f3460; color: #e0e0e0; border: 1px solid #533483; border-radius: 4px; font-size: 13px; }
.btn-schedule:hover { background: #1e2d50; }
</style>
