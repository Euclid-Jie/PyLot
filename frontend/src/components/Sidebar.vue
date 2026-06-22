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
        <button class="btn-footer" :class="{ active: store.currentView === 'services' }" @click="store.setView('services')">🖥 服务</button>
        <button class="btn-footer" :class="{ active: store.currentView === 'schedule' }" @click="store.setView('schedule')">📅</button>
        <button class="btn-footer btn-settings" :class="{ active: store.currentView === 'settings' }" @click="store.setView('settings')">⚙️</button>
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
  store.setScript(0)
}

function newWorkflow() {
  store.newWorkflow()
}
</script>

<style scoped>
.sidebar-inner { display: flex; flex-direction: column; height: 100%; }

.sidebar-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 16px 16px 8px;
  font-size: 12px; font-weight: 600; color: var(--text-muted);
  letter-spacing: .04em; text-transform: uppercase;
}
.wf-header-section { margin-top: 4px; border-top: 1px solid var(--border); padding-top: 16px; }

.btn-add {
  width: 22px; height: 22px; border-radius: var(--radius-sm); border: none;
  background: transparent; color: var(--text-muted);
  font-size: 18px; line-height: 1; display: flex; align-items: center; justify-content: center;
  transition: background .12s, color .12s;
}
.btn-add:hover { background: var(--surface2); color: var(--text); }

.cat-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 5px 16px; font-size: 11px; font-weight: 600; letter-spacing: .04em; text-transform: uppercase;
  color: var(--text-muted); background: var(--surface2);
  cursor: pointer; user-select: none; transition: background .12s, color .12s;
  border-top: 1px solid var(--border);
}
.cat-header:hover { background: var(--surface); color: var(--text-dim); }

.cat-scripts { padding: 2px 0 6px; }

.script-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 7px 16px 7px 20px;
  font-size: 14px; cursor: pointer; color: var(--text-dim);
  border-left: 2px solid transparent;
  transition: background .1s, color .1s, border-color .1s;
  white-space: nowrap;
}
.script-item:hover { background: var(--surface); color: var(--text); }
.script-item.active {
  background: var(--accent-dim); color: var(--accent);
  border-left-color: var(--accent); font-weight: 500;
}

.script-name { overflow: hidden; text-overflow: ellipsis; }
.spinner { animation: spin 1s linear infinite; display: inline-block; color: var(--green); font-size: 13px; flex-shrink: 0; margin-left: 4px; }
@keyframes spin { to { transform: rotate(360deg); } }

.sidebar-footer { margin-top: auto; padding: 12px; border-top: 1px solid var(--border); }
.footer-row { display: flex; gap: 6px; }
.btn-footer {
  flex: 1; padding: 7px 8px;
  background: transparent; color: var(--text-dim);
  border: 1px solid var(--border); border-radius: var(--radius);
  font-size: 13px; font-weight: 500;
  transition: background .12s, color .12s, border-color .12s;
}
.btn-footer:hover { background: var(--surface2); color: var(--text); border-color: var(--text-muted); }
.btn-footer.active { background: var(--accent-dim); color: var(--accent); border-color: var(--accent); }
.btn-settings { flex: 0 0 36px; }
</style>
