<template>
  <div class="schedule-view">
    <div class="view-header">
      <h2>定时任务总览</h2>
      <button class="btn-add" @click="openAdd">新增定时</button>
    </div>
    <table v-if="overview.length">
      <thead>
        <tr>
          <th>脚本名称</th>
          <th>Cron 表达式</th>
          <th>下次运行时间</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in overview" :key="item.scheduleId" :class="item.scriptId < 0 ? 'row-wf' : 'row-script'">
          <td>
            <div class="name-cell">
              <span class="type-dot" :class="item.scriptId < 0 ? 'wf' : 'sc'"></span>
              {{ item.scriptId < 0 ? (item.scriptName || '工作流') : item.scriptName }}
            </div>
          </td>
          <td><code>{{ item.cronExpr }}</code></td>
          <td>{{ fmtTime(item.nextRun, item.enabled) }}</td>
          <td>
            <button :class="['btn-toggle', item.enabled ? 'on' : 'off']" @click="toggle(item)">
              {{ item.enabled ? '启用' : '禁用' }}
            </button>
          </td>
          <td>
            <div class="actions">
              <button class="btn-edit" @click="openEdit(item)">编辑</button>
              <button class="btn-del" @click="del(item.scheduleId)">删除</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无定时任务，请在脚本配置页添加</div>
    <div v-if="errorMsg" class="toast-error">{{ errorMsg }}</div>
    <TimerModal
      v-if="showTimer"
      :script-id="timerScriptId"
      :schedule="editingSchedule"
      :allow-target-select="true"
      :existing-schedules="overview"
      @saved="load"
      @close="closeTimer"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { GetScheduleOverview, ToggleSchedule, DeleteSchedule } from '../../wailsjs/go/main/App.js'
import TimerModal from './TimerModal.vue'

const overview = ref([])
const errorMsg = ref('')
const showTimer = ref(false)
const editingSchedule = ref(null)
const timerScriptId = ref(0)
let timer = null

onMounted(async () => {
  await load()
  timer = setInterval(load, 60000)
})
onUnmounted(() => clearInterval(timer))

async function load() {
  overview.value = await GetScheduleOverview() || []
}

function fmtTime(t, enabled) {
  if (!enabled) return '—'
  if (!t) return '—'
  const dt = new Date(t)
  if (Number.isNaN(dt.getTime()) || dt.getFullYear() <= 1) return '—'
  return dt.toLocaleString()
}

function openAdd() {
  editingSchedule.value = null
  timerScriptId.value = 0
  showTimer.value = true
}

function openEdit(item) {
  editingSchedule.value = {
    id: item.scheduleId,
    scheduleId: item.scheduleId,
    scriptId: item.scriptId,
    cronExpr: item.cronExpr,
    enabled: item.enabled,
  }
  timerScriptId.value = item.scriptId
  showTimer.value = true
}

async function closeTimer() {
  showTimer.value = false
  editingSchedule.value = null
  timerScriptId.value = 0
  await load()
}

async function toggle(item) {
  try {
    errorMsg.value = ''
    await ToggleSchedule(item.scheduleId, !item.enabled)
    await load()
  } catch (err) {
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  }
}

async function del(id) {
  if (!confirm('确认删除此定时任务？')) return
  try {
    errorMsg.value = ''
    await DeleteSchedule(id)
    await load()
  } catch (err) {
    errorMsg.value = formatError(err)
    setTimeout(() => { errorMsg.value = '' }, 3000)
  }
}

function formatError(err) {
  const message = err?.message || String(err || '操作失败')
  return message.replace(/^Error:\s*/, '')
}
</script>

<style scoped>
.schedule-view { padding: 0; }
.view-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); }
h2 { font-size: 20px; font-weight: 600; }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th { text-align: left; padding: 8px 14px; font-size: 12px; font-weight: 600; letter-spacing: .04em; text-transform: uppercase; color: var(--text-muted); border-bottom: 1px solid var(--border); }
td { padding: 11px 14px; border-bottom: 1px solid var(--border); vertical-align: middle; color: var(--text-dim); }
td:first-child { color: var(--text); }
.name-cell { display: flex; align-items: center; gap: 10px; }
.type-dot { width: 2px; height: 18px; border-radius: 2px; flex-shrink: 0; }
.type-dot.sc { background: var(--accent); }
.type-dot.wf { background: var(--orange); }
code { background: var(--surface); padding: 3px 8px; border-radius: var(--radius-sm); color: var(--green); font-size: 13px; border: 1px solid var(--border); font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace; }
.btn-toggle { padding: 4px 14px; border-radius: 20px; font-size: 13px; font-weight: 500; cursor: pointer; transition: opacity .12s; }
.btn-toggle.on  { background: var(--orange-dim); color: var(--orange); border: 1px solid rgba(210,153,34,.4); }
.btn-toggle.off { background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); }
.btn-toggle:hover { opacity: .8; }
.actions { display: flex; gap: 8px; align-items: center; }
.btn-add, .btn-edit, .btn-del { border-radius: var(--radius); font-size: 13px; transition: color .12s, border-color .12s, background .12s; }
.btn-add { padding: 6px 14px; background: var(--accent); color: #fff; border: 1px solid var(--accent); font-weight: 500; }
.btn-add:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.btn-edit { background: var(--surface2); color: var(--text-dim); border: 1px solid var(--border); padding: 4px 12px; }
.btn-edit:hover { color: var(--text); border-color: var(--text-muted); }
.btn-del { background: transparent; color: var(--text-muted); border: 1px solid var(--border); padding: 4px 12px; }
.btn-del:hover { color: var(--red); border-color: var(--red); }
.empty { color: var(--text-muted); text-align: center; margin-top: 80px; font-size: 14px; }
.toast-error { position: fixed; right: 24px; bottom: 280px; max-width: 360px; background: var(--red-dim); color: var(--red); border: 1px solid rgba(248,81,73,.32); border-radius: var(--radius); padding: 9px 12px; font-size: 13px; box-shadow: 0 8px 24px rgba(0,0,0,.25); z-index: 200; }
</style>
