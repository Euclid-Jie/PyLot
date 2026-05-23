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
        <tr v-for="item in overview" :key="item.scheduleId">
          <td>{{ item.scriptName }}</td>
          <td><code>{{ item.cronExpr }}</code></td>
          <td>{{ fmtTime(item.nextRun) }}</td>
          <td>
            <label class="toggle">
              <input type="checkbox" :checked="item.enabled" @change="toggle(item)" />
              <span>{{ item.enabled ? '启用' : '禁用' }}</span>
            </label>
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
th { text-align: left; padding: 8px 12px; background: #0f3460; color: #aaa; font-weight: normal; }
td { padding: 8px 12px; border-bottom: 1px solid #222; }
code { background: #252526; padding: 2px 6px; border-radius: 3px; color: #4caf50; }
.toggle { display: flex; align-items: center; gap: 6px; cursor: pointer; }
.toggle input { cursor: pointer; }
.btn-del { background: #3a3a3a; color: #e74c3c; border: 1px solid #555; padding: 3px 10px; border-radius: 4px; font-size: 12px; }
.empty { color: #666; text-align: center; margin-top: 60px; font-size: 14px; }
</style>
