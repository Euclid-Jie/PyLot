<template>
  <div class="sidebar-inner">
    <div class="brand">
      <div class="brand-mark">P</div>
      <div><strong>PyLot</strong><span>脚本调度中心</span></div>
    </div>

    <nav class="primary-nav" aria-label="主要导航">
      <button :class="{ active: store.currentView === 'schedule' }" @click="store.setView('schedule')"><UiIcon name="clock" /><span>定时任务</span></button>
      <button :class="{ active: store.currentView === 'services' }" @click="store.setView('services')"><UiIcon name="server" /><span>服务</span></button>
    </nav>

    <div class="resource-header"><span>资源</span><button class="ui-icon-btn" aria-label="新建脚本" title="新建脚本" @click="newScript"><UiIcon name="add" /></button></div>
    <div class="search-box">
      <UiIcon name="search" :size="15" />
      <input v-model.trim="query" aria-label="搜索脚本或工作流" placeholder="搜索脚本或工作流" />
      <button v-if="query" aria-label="清除搜索" title="清除搜索" @click="query = ''"><UiIcon name="x" :size="13" /></button>
    </div>

    <div class="resource-scroll">
      <section class="resource-section">
        <button class="section-title" @click="scriptsOpen = !scriptsOpen"><UiIcon :name="scriptsOpen ? 'chevronDown' : 'chevronRight'" :size="14" /><span>脚本</span><strong>{{ filteredScripts.length }}</strong></button>
        <div v-if="scriptsOpen">
          <div v-for="category in visibleCategories" :key="category.key" class="category">
            <button class="category-title" @click="category.open = !category.open"><UiIcon :name="category.open ? 'chevronDown' : 'chevronRight'" :size="13" /><span>{{ category.label }}</span><strong>{{ scriptsByCategory(category.key).length }}</strong></button>
            <div v-if="category.open">
              <button v-for="script in scriptsByCategory(category.key)" :key="script.id" class="resource-item" :class="{ active: store.selectedScriptID === script.id && store.currentView === 'script' }" :title="script.name" @click="store.setScript(script.id)">
                <UiIcon name="file" :size="15" /><span>{{ script.name }}</span><i v-if="store.runningScripts.has(script.id)" class="running-dot"></i>
              </button>
            </div>
          </div>
          <div v-if="!filteredScripts.length" class="resource-empty">没有匹配的脚本</div>
        </div>
      </section>

      <section class="resource-section">
        <div class="section-title-row">
          <button class="section-title" @click="workflowsOpen = !workflowsOpen"><UiIcon :name="workflowsOpen ? 'chevronDown' : 'chevronRight'" :size="14" /><span>工作流</span><strong>{{ filteredWorkflows.length }}</strong></button>
          <button class="ui-icon-btn compact" aria-label="新建工作流" title="新建工作流" @click="newWorkflow"><UiIcon name="add" /></button>
        </div>
        <div v-if="workflowsOpen">
          <button v-for="workflow in filteredWorkflows" :key="workflow.id" class="resource-item" :class="{ active: store.selectedWorkflowId === workflow.id && store.currentView === 'workflow' }" :title="workflow.name" @click="store.openWorkflow(workflow.id)">
            <UiIcon name="flow" :size="15" /><span>{{ workflow.name }}</span><i v-if="store.runningScripts.has(-workflow.id)" class="running-dot"></i>
          </button>
          <div v-if="!filteredWorkflows.length" class="resource-empty">没有匹配的工作流</div>
        </div>
      </section>
    </div>

    <footer><button :class="{ active: store.currentView === 'settings' }" @click="store.setView('settings')"><UiIcon name="settings" /><span>设置</span></button></footer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { GetScripts, GetWorkflows } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const allScripts = ref([])
const allWorkflows = ref([])
const query = ref('')
const scriptsOpen = ref(true)
const workflowsOpen = ref(true)
const categories = ref([
  { key: 'crawler', label: '数据爬取上传', open: true },
  { key: 'processor', label: '数据处理', open: true },
  { key: 'tool', label: '个人工具', open: true },
])

const normalizedQuery = computed(() => query.value.toLocaleLowerCase())
const filteredScripts = computed(() => allScripts.value.filter(item => item.name.toLocaleLowerCase().includes(normalizedQuery.value)))
const filteredWorkflows = computed(() => allWorkflows.value.filter(item => item.name.toLocaleLowerCase().includes(normalizedQuery.value)))
const visibleCategories = computed(() => categories.value.filter(category => scriptsByCategory(category.key).length || !query.value))

onMounted(loadResources)
watch(() => store.scriptListVersion, loadResources)
async function loadResources() {
  const [scripts, workflows] = await Promise.all([GetScripts(), GetWorkflows()])
  allScripts.value = scripts || []
  allWorkflows.value = workflows || []
}
function scriptsByCategory(category) { return filteredScripts.value.filter(script => script.category === category) }
function newScript() { store.setScript(0) }
function newWorkflow() { store.newWorkflow() }
</script>

<style scoped>
.sidebar-inner { height: 100%; display: flex; flex-direction: column; }
.brand { min-height: 66px; display: flex; align-items: center; gap: 11px; padding: 12px 14px; border-bottom: 1px solid var(--border); }
.brand-mark { width: 34px; height: 34px; display: grid; place-items: center; flex-shrink: 0; border-radius: 8px; background: var(--accent); color: #fff; font-size: 17px; font-weight: var(--weight-bold); }
.brand strong, .brand span { display: block; }
.brand strong { color: var(--text); font-size: var(--type-section-title); font-weight: var(--weight-semibold); }
.brand span { margin-top: 1px; color: var(--text-muted); font-size: var(--type-label); }
.primary-nav { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; padding: 10px 10px 4px; }
.primary-nav button, footer button { min-height: 36px; display: flex; align-items: center; gap: 8px; padding: 0 11px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); font-size: var(--type-body); font-weight: var(--weight-regular); text-align: left; }
.primary-nav button:hover, footer button:hover { background: var(--surface-hover); color: var(--text); }
.primary-nav button.active, footer button.active { border-color: rgba(59, 130, 246, .28); background: var(--accent-dim); color: var(--accent); font-weight: var(--weight-medium); }
.resource-header { display: flex; align-items: center; justify-content: space-between; padding: 13px 12px 7px 14px; color: var(--text-muted); font-size: var(--type-label); font-weight: var(--weight-medium); }
.resource-header .ui-icon-btn, .compact { width: 26px; height: 26px; }
.search-box { height: 32px; display: flex; align-items: center; gap: 7px; margin: 0 10px 8px; padding: 0 9px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--text-muted); }
.search-box:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.search-box input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--text); font-size: var(--type-label); }
.search-box button { display: grid; place-items: center; border: 0; background: transparent; color: var(--text-muted); }
.resource-scroll { min-height: 0; flex: 1; overflow-y: auto; padding: 0 8px 12px; }
.resource-section + .resource-section { margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border); }
.section-title-row { display: flex; align-items: center; }
.section-title-row .section-title { flex: 1; }
.section-title, .category-title { width: 100%; display: grid; grid-template-columns: 16px minmax(0, 1fr) auto; align-items: center; gap: 5px; border: 0; background: transparent; color: var(--text-dim); text-align: left; }
.section-title { min-height: 30px; padding: 0 6px; font-size: var(--type-label); font-weight: var(--weight-medium); }
.category-title { min-height: 28px; padding: 0 8px 0 16px; color: var(--text-muted); font-size: var(--type-label); }
.section-title:hover, .category-title:hover { color: var(--text); }
.section-title strong, .category-title strong { color: var(--text-muted); font-size: var(--type-caption); font-weight: var(--weight-medium); }
.resource-item { width: 100%; min-height: 32px; display: grid; grid-template-columns: 18px minmax(0, 1fr) 10px; align-items: center; gap: 7px; margin: 1px 0; padding: 0 8px 0 20px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); font-size: var(--type-body); font-weight: var(--weight-regular); text-align: left; }
.resource-item span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.resource-item:hover { background: var(--surface-hover); color: var(--text); }
.resource-item.active { border-color: rgba(59, 130, 246, .22); background: var(--accent-dim); color: var(--accent); font-weight: var(--weight-medium); }
.running-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--green); box-shadow: 0 0 0 3px var(--green-dim); animation: pulse 1.5s ease-in-out infinite; }
.resource-empty { padding: 10px 20px; color: var(--text-muted); font-size: var(--type-label); }
footer { padding: 9px 10px; border-top: 1px solid var(--border); }
footer button { width: 100%; }
@keyframes pulse { 50% { opacity: .45; } }
</style>
