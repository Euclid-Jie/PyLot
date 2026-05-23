<template>
  <div class="sidebar-inner">
    <!-- 脚本管理 header -->
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

    <!-- 工作流管理 header -->
    <div class="sidebar-header wf-header-section">
      <span>工作流</span>
      <button class="btn-add" @click="newWorkflow">+</button>
    </div>

    <div class="category">
      <div class="cat-scripts">
        <div
          v-for="w in allWorkflows"
          :key="'wf-' + w.id"
          class="script-item"
          :class="{ active: store.selectedWorkflowId === w.id && store.currentView === 'workflow' }"
          @click="store.openWorkflow(w.id)"
        >
          <span class="script-name">{{ w.name }}</span>
          <span v-if="store.runningScripts.has(-w.id)" class="spinner">⟳</span>
        </div>
      </div>
    </div>

    <div class="sidebar-footer">
      <div class="footer-row">
        <button class="btn-footer" @click="store.setView('schedule')">📅 Schedule</button>
        <button class="btn-footer btn-settings" @click="store.setView('settings')">⚙️</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { GetScripts, GetWorkflows } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'

const store = useMainStore()
const allScripts = ref([])
const allWorkflows = ref([])

const categories = ref([
  { key: 'crawler', label: '数据爬取上传', icon: '🕷', open: true },
  { key: 'processor', label: '数据处理', icon: '⚙️', open: true },
  { key: 'tool', label: '个人工具', icon: '🔧', open: true },
])

onMounted(async () => {
  await loadScripts()
  allWorkflows.value = await GetWorkflows() || []
  const cfg = await GetGlobalConfig()
  envPath.value = cfg.envFilePath || ''
})

watch(() => store.scriptListVersion, async () => {
  await loadScripts()
  allWorkflows.value = await GetWorkflows() || []
})

async function loadScripts() {
  allScripts.value = await GetScripts() || []
}

function scriptsByCategory(cat) {
  return allScripts.value.filter(s => s.category === cat)
}

function newScript() {
  store.setScript(-1)
}

function newWorkflow() {
  store.selectedWorkflowId = null
  store.setView('workflow')
}
</script>

<style scoped>
.sidebar-inner { display: flex; flex-direction: column; height: 100%; }
.sidebar-header { display: flex; justify-content: space-between; align-items: center; padding: 10px 12px; border-bottom: 1px solid var(--border); font-size: 13px; font-weight: bold; color: var(--text); }
.wf-header-section { border-top: 2px solid var(--border); margin-top: 4px; }
.btn-add { background: #533483; color: #fff; border: none; border-radius: 4px; width: 24px; height: 24px; font-size: 16px; line-height: 1; cursor: pointer; }
.category { border-bottom: 1px solid var(--border); }
.cat-header { display: flex; justify-content: space-between; padding: 8px 12px; font-size: 12px; color: var(--text-muted); cursor: pointer; user-select: none; }
.cat-header:hover { background: var(--surface); }
.cat-scripts { padding: 2px 0; }
.script-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 16px; font-size: 13px; cursor: pointer; color: var(--text); }
.script-item:hover { background: var(--surface); }
.script-item.active { background: var(--accent); color: #fff; }
.script-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.spinner { animation: spin 1s linear infinite; display: inline-block; color: #4caf50; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.sidebar-footer { margin-top: auto; padding: 10px 12px; border-top: 1px solid var(--border); }
.footer-row { display: flex; gap: 6px; }
.btn-footer { flex: 1; padding: 7px; background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 4px; font-size: 13px; }
.btn-footer:hover { background: var(--surface2); }
.btn-settings { flex: 0 0 36px; color: var(--text-muted); }
</style>
