<template>
  <div class="schedule-view">
    <h2>定时任务总览</h2>
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
          <td>{{ fmtTime(item.nextRun) }}</td>
          <td>
            <button :class="['btn-toggle', item.enabled ? 'on' : 'off']" @click="toggle(item)">
              {{ item.enabled ? '启用' : '禁用' }}
            </button>
          </td>
          <td><button class="btn-del" @click="del(item.scheduleId)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无定时任务，请在脚本配置页添加</div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { GetScheduleOverview, ToggleSchedule, DeleteSchedule } from '../../wailsjs/go/main/App.js'

const overview = ref([])
let timer = null

onMounted(async () => {
  await load()
  timer = setInterval(load, 60000)
})
onUnmounted(() => clearInterval(timer))

async function load() {
  overview.value = await GetScheduleOverview() || []
}

function fmtTime(t) {
  if (!t) return '—'
  return new Date(t).toLocaleString()
}

async function toggle(item) {
  await ToggleSchedule(item.scheduleId, !item.enabled)
  await load()
}

async function del(id) {
  if (!confirm('确认删除此定时任务？')) return
  await DeleteSchedule(id)
  await load()
}
</script>

<style scoped>
.schedule-view { padding: 0; }
h2 { font-size: 20px; font-weight: 600; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); }
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
.btn-del { background: transparent; color: var(--text-muted); border: 1px solid var(--border); padding: 4px 12px; border-radius: var(--radius); font-size: 13px; transition: color .12s, border-color .12s; }
.btn-del:hover { color: var(--red); border-color: var(--red); }
.empty { color: var(--text-muted); text-align: center; margin-top: 80px; font-size: 14px; }
</style>
