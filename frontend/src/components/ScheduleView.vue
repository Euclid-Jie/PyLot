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
.schedule-view { padding: 4px; }
h2 { margin-bottom: 16px; font-size: 16px; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: center; padding: 8px 12px; background: var(--surface); color: var(--text-muted); font-weight: normal; }
th:first-child { text-align: left; }
td { padding: 8px 12px; border-bottom: 1px solid var(--border); text-align: center; vertical-align: middle; }
td:first-child { text-align: left; }
.name-cell { display: flex; align-items: center; gap: 8px; }
.type-dot { width: 3px; height: 18px; border-radius: 2px; flex-shrink: 0; }
.type-dot.sc { background: var(--accent); }
.type-dot.wf { background: #e65100; }
code { background: #252526; padding: 2px 6px; border-radius: 3px; color: #4caf50; }
.btn-toggle { padding: 3px 12px; border-radius: 12px; border: none; font-size: 12px; cursor: pointer; font-weight: 500; }
.btn-toggle.on  { background: #e65100; color: #fff; }
.btn-toggle.off { background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); }
.btn-del { background: #3a3a3a; color: #e74c3c; border: 1px solid #555; padding: 3px 10px; border-radius: 4px; font-size: 12px; }
.empty { color: #666; text-align: center; margin-top: 60px; font-size: 14px; }
</style>
