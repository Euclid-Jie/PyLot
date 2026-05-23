<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>定时规则配置</h3>
      <div class="form-row">
        <label>定时类型</label>
        <select v-model="timerType">
          <option value="daily">每日</option>
          <option value="weekly">每周</option>
        </select>
      </div>
      <div class="form-row" v-if="timerType === 'weekly'">
        <label>星期</label>
        <div class="weekdays">
          <label v-for="(d, i) in weekdays" :key="i">
            <input type="checkbox" :value="i+1" v-model="selectedDays" /> {{ d }}
          </label>
        </div>
      </div>
      <div class="form-row">
        <label>时间</label>
        <input type="time" v-model="timeVal" />
      </div>
      <div class="form-row">
        <label>Cron 预览</label>
        <code>{{ cronPreview }}</code>
      </div>
      <div class="form-row">
        <label>启用</label>
        <input type="checkbox" v-model="enabled" />
      </div>
      <div class="modal-actions">
        <button class="btn-primary" @click="handleSave">保存</button>
        <button class="btn-cancel" @click="$emit('close')">取消</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { SaveSchedule } from '../../wailsjs/go/main/App.js'

const props = defineProps({ scriptId: Number })
const emit = defineEmits(['close'])

const timerType = ref('daily')
const timeVal = ref('08:00')
const selectedDays = ref([1])
const enabled = ref(true)
const weekdays = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

const cronPreview = computed(() => {
  const [h, m] = timeVal.value.split(':')
  if (timerType.value === 'daily') return `${m} ${h} * * *`
  const days = selectedDays.value.sort().join(',')
  return `${m} ${h} * * ${days}`
})

async function handleSave() {
  await SaveSchedule({
    id: 0,
    scriptId: props.scriptId,
    cronExpr: cronPreview.value,
    enabled: enabled.value ? 1 : 0,
  })
  emit('close')
}
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: #16213e; border: 1px solid #0f3460; border-radius: 8px; padding: 24px; min-width: 380px; }
h3 { margin-bottom: 16px; }
.form-row { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.form-row label { width: 90px; font-size: 13px; color: #aaa; flex-shrink: 0; }
.form-row select, .form-row input[type=time] { padding: 5px 8px; background: #252526; border: 1px solid #444; color: #e0e0e0; border-radius: 4px; }
code { background: #252526; padding: 4px 8px; border-radius: 3px; font-size: 13px; color: #4caf50; }
.weekdays { display: flex; gap: 8px; flex-wrap: wrap; }
.weekdays label { font-size: 13px; display: flex; align-items: center; gap: 4px; }
.modal-actions { display: flex; gap: 8px; margin-top: 16px; }
.btn-primary { background: #2196f3; color: #fff; border: none; padding: 7px 16px; border-radius: 4px; }
.btn-cancel { background: #555; color: #ccc; border: none; padding: 7px 16px; border-radius: 4px; }
</style>
