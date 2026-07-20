<template>
  <aside class="script-list-pane">
    <header>
      <div>
        <h2>{{ currentListName }}</h2>
        <span>{{ visibleScripts.length }} 个脚本</span>
      </div>
      <button class="ui-icon-btn add-script" title="在当前列表新建脚本" aria-label="在当前列表新建脚本" @click="newScript">
        <UiIcon name="add" />
      </button>
    </header>

    <div class="script-search">
      <UiIcon name="search" :size="15" />
      <input v-model.trim="query" aria-label="搜索当前列表的脚本" placeholder="搜索当前列表" />
      <button v-if="query" title="清除搜索" aria-label="清除搜索" @click="query = ''"><UiIcon name="x" :size="13" /></button>
    </div>

    <div class="script-rows">
      <button
        v-for="script in visibleScripts"
        :key="script.id"
        class="script-row"
        :class="{ active: store.selectedScriptID === script.id }"
        @click="store.setScript(script.id)"
      >
        <span class="file-mark"><UiIcon name="file" :size="15" /></span>
        <span class="script-row-main">
          <strong>{{ script.name }}</strong>
          <small>{{ script.launchMode || 'script' }}<template v-if="workDirName(script)"> · {{ workDirName(script) }}</template></small>
        </span>
        <i v-if="store.runningScripts.has(script.id)" class="running-dot" title="运行中"></i>
      </button>
      <div v-if="!visibleScripts.length" class="empty-list">
        <UiIcon name="file" :size="22" />
        <span>{{ query ? '没有匹配的脚本' : '当前列表暂无脚本' }}</span>
        <button v-if="!query" class="ui-btn small" @click="newScript"><UiIcon name="add" />新增脚本</button>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { GetScriptLists, GetScripts } from '../../wailsjs/go/main/App.js'
import { useMainStore } from '../stores/main.js'
import UiIcon from './UiIcon.vue'

const store = useMainStore()
const scripts = ref([])
const lists = ref([])
const query = ref('')

const currentListName = computed(() => {
  if (store.selectedScriptListId === 'all') return '全部脚本'
  if (store.selectedScriptListId === 'unassigned') return '未分类'
  return lists.value.find(item => item.id === store.selectedScriptListId)?.name || '脚本列表'
})
const filteredByList = computed(() => {
  if (store.selectedScriptListId === 'all') return scripts.value
  if (store.selectedScriptListId === 'unassigned') return scripts.value.filter(item => !item.listId)
  return scripts.value.filter(item => item.listId === store.selectedScriptListId)
})
const visibleScripts = computed(() => {
  const normalized = query.value.toLocaleLowerCase()
  return filteredByList.value.filter(item => item.name.toLocaleLowerCase().includes(normalized))
})

onMounted(load)
watch(() => store.scriptListVersion, load)
watch(() => store.selectedScriptListId, () => { query.value = '' })

async function load() {
  const [scriptRows, listRows] = await Promise.all([GetScripts(), GetScriptLists()])
  scripts.value = scriptRows || []
  lists.value = listRows || []
}

function newScript() { store.setScript(0) }
function workDirName(script) {
  const value = (script.workDir || '').replace(/[\\/]+$/, '')
  return value.split(/[\\/]/).pop() || ''
}
</script>

<style scoped>
.script-list-pane { width: 270px; min-width: 230px; flex: 0 0 270px; display: flex; flex-direction: column; overflow: hidden; border-right: 1px solid var(--border); background: var(--sidebar-bg); }
header { min-height: 66px; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--border); }
header h2 { overflow: hidden; color: var(--text); font-size: var(--type-section-title); font-weight: var(--weight-semibold); text-overflow: ellipsis; white-space: nowrap; }
header span { display: block; margin-top: 2px; color: var(--text-muted); font-size: var(--type-caption); }
.add-script { border-color: var(--border); background: var(--surface); color: var(--text-dim); }
.script-search { height: 34px; display: flex; align-items: center; gap: 7px; margin: 10px; padding: 0 9px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--text-muted); }
.script-search:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.script-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--text); font-size: var(--type-label); }
.script-search button { display: grid; place-items: center; border: 0; background: transparent; color: var(--text-muted); }
.script-rows { min-height: 0; flex: 1; overflow-y: auto; padding: 0 8px 12px; }
.script-row { width: 100%; min-height: 52px; display: grid; grid-template-columns: 20px minmax(0, 1fr) 10px; align-items: center; gap: 8px; padding: 7px 9px; border: 1px solid transparent; border-radius: var(--radius); background: transparent; color: var(--text-dim); text-align: left; }
.script-row:hover { background: var(--surface-hover); color: var(--text); }
.script-row.active { border-color: rgba(59, 130, 246, .24); background: var(--accent-dim); color: var(--accent); }
.file-mark { align-self: flex-start; margin-top: 3px; color: var(--text-muted); }
.script-row.active .file-mark { color: var(--accent); }
.script-row-main { min-width: 0; display: flex; flex-direction: column; gap: 3px; }
.script-row-main strong, .script-row-main small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.script-row-main strong { font-size: var(--type-body); font-weight: var(--weight-medium); }
.script-row-main small { color: var(--text-muted); font-size: var(--type-caption); }
.running-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--green); box-shadow: 0 0 0 3px var(--green-dim); animation: pulse 1.5s ease-in-out infinite; }
.empty-list { min-height: 180px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 9px; padding: 20px; color: var(--text-muted); font-size: var(--type-label); text-align: center; }
@keyframes pulse { 50% { opacity: .45; } }
</style>
