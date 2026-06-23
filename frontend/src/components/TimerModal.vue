<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <section class="schedule-modal" role="dialog" aria-modal="true" aria-labelledby="timer-title">
      <header class="modal-header">
        <div>
          <div class="eyebrow">Schedule</div>
          <h3 id="timer-title">{{ isEditing ? '编辑定时规则' : '定时规则配置' }}</h3>
        </div>
        <button class="icon-btn" title="关闭" @click="emit('close')">×</button>
      </header>

      <div class="modal-body">
        <div v-if="allowTargetSelect" class="field">
          <label>目标</label>
          <select v-model.number="selectedScriptId">
            <option :value="0" disabled>选择脚本或工作流</option>
            <optgroup v-if="scripts.length" label="脚本">
              <option v-for="script in scripts" :key="script.id" :value="script.id">{{ script.name }}</option>
            </optgroup>
            <optgroup v-if="workflows.length" label="工作流">
              <option v-for="workflow in workflows" :key="workflow.id" :value="-workflow.id">{{ workflow.name }}</option>
            </optgroup>
          </select>
        </div>

        <div class="field">
          <label>配置方式</label>
          <div class="segmented">
            <button :class="{ active: scheduleMode === 'preset' }" @click="scheduleMode = 'preset'">快捷</button>
            <button :class="{ active: scheduleMode === 'custom' }" @click="scheduleMode = 'custom'">Cron</button>
          </div>
        </div>

        <template v-if="scheduleMode === 'preset'">
          <div class="field">
            <label>规则</label>
            <select v-model="presetType">
              <option value="daily_once">每日一次</option>
              <option value="daily_multi">每天多时刻</option>
              <option value="weekly">每周</option>
              <option value="workday">工作日</option>
              <option value="interval">循环间隔</option>
            </select>
          </div>

          <div v-if="presetType === 'daily_once' || presetType === 'workday'" class="field">
            <label>时间</label>
            <input type="time" v-model="timeVal" />
          </div>

          <div v-if="presetType === 'daily_multi'" class="field field-top">
            <label>时刻</label>
            <div class="time-stack">
              <div v-for="(time, index) in multiTimes" :key="`${time}-${index}`" class="time-line">
                <input type="time" v-model="multiTimes[index]" />
                <button class="ghost-btn danger" title="移除" :disabled="multiTimes.length === 1" @click="removeTime(index)">×</button>
              </div>
              <button class="ghost-btn add-time" @click="addTime">添加时刻</button>
            </div>
          </div>

          <div v-if="presetType === 'weekly'" class="field field-top">
            <label>星期</label>
            <div class="weekday-grid">
              <label v-for="day in weekdays" :key="day.value" :class="{ checked: selectedDays.includes(day.value) }">
                <input type="checkbox" :value="day.value" v-model="selectedDays" />
                <span>{{ day.label }}</span>
              </label>
            </div>
          </div>

          <div v-if="presetType === 'weekly'" class="field">
            <label>时间</label>
            <input type="time" v-model="timeVal" />
          </div>

          <div v-if="presetType === 'interval'" class="field">
            <label>间隔</label>
            <div class="inline-controls">
              <input class="number-input" type="number" min="1" :max="intervalLimit" v-model.number="intervalEvery" />
              <select v-model="intervalUnit">
                <option value="minutes">分钟</option>
                <option value="hours">小时</option>
              </select>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="field">
            <label>表达式</label>
            <input class="cron-input" v-model.trim="customCron" placeholder="0 8 * * *" />
          </div>
          <div class="field field-top">
            <label>常用</label>
            <div class="preset-chips">
              <button v-for="item in customPresets" :key="item.expr" @click="customCron = item.expr">{{ item.label }}</button>
            </div>
          </div>
        </template>

        <div class="preview-block">
          <div class="preview-title">
            <span>Cron 预览</span>
            <strong v-if="generatedCrons.length > 1">{{ generatedCrons.length }} 条</strong>
          </div>
          <div v-if="generatedCrons.length" class="cron-list">
            <code v-for="expr in generatedCrons" :key="expr">{{ expr }}</code>
          </div>
          <div v-else class="empty-preview">—</div>
        </div>

        <div class="field compact">
          <label>启用</label>
          <label class="switch">
            <input type="checkbox" v-model="enabled" />
            <span></span>
          </label>
        </div>

        <div v-if="displayError" class="message error">{{ displayError }}</div>
        <div v-else-if="displayWarning" class="message warning">{{ displayWarning }}</div>
        <div v-if="toast" class="message success">{{ toast }}</div>
      </div>

      <footer class="modal-actions">
        <button class="btn-cancel" @click="emit('close')">取消</button>
        <button class="btn-primary" :disabled="saving || !!validationError" @click="handleSave">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { GetScripts, GetWorkflows, SaveSchedule } from '../../wailsjs/go/main/App.js'

const props = defineProps({
  scriptId: { type: Number, default: 0 },
  schedule: { type: Object, default: null },
  allowTargetSelect: { type: Boolean, default: false },
  existingSchedules: { type: Array, default: () => [] },
})
const emit = defineEmits(['close', 'saved'])

const scripts = ref([])
const workflows = ref([])
const selectedScriptId = ref(props.schedule?.scriptId ?? props.scriptId ?? 0)
const scheduleMode = ref('preset')
const presetType = ref('daily_once')
const timeVal = ref('08:00')
const multiTimes = ref(['08:00', '12:00'])
const selectedDays = ref([1])
const intervalEvery = ref(1)
const intervalUnit = ref('hours')
const customCron = ref('0 8 * * *')
const enabled = ref(true)
const saving = ref(false)
const toast = ref('')
const actionError = ref('')

const isEditing = computed(() => !!props.schedule?.scheduleId || !!props.schedule?.id)
const scheduleId = computed(() => props.schedule?.scheduleId || props.schedule?.id || 0)
const allowTargetSelect = computed(() => props.allowTargetSelect)

const weekdays = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 0, label: '周日' },
]

const customPresets = [
  { label: '每 15 分钟', expr: '*/15 * * * *' },
  { label: '每小时', expr: '0 * * * *' },
  { label: '工作日 09:00', expr: '0 9 * * 1-5' },
  { label: '每日 08/12/18 点', expr: '0 8,12,18 * * *' },
]

const intervalLimit = computed(() => intervalUnit.value === 'minutes' ? 59 : 23)

const generatedCrons = computed(() => {
  if (scheduleMode.value === 'custom') {
    const expr = normalizeCron(customCron.value)
    return expr ? [expr] : []
  }

  if (presetType.value === 'daily_once') {
    return [cronForTime(timeVal.value)]
  }

  if (presetType.value === 'daily_multi') {
    return uniqueTimes(multiTimes.value).map(time => cronForTime(time))
  }

  if (presetType.value === 'weekly') {
    return [cronForTime(timeVal.value, sortedDays(selectedDays.value).join(','))]
  }

  if (presetType.value === 'workday') {
    return [cronForTime(timeVal.value, '1-5')]
  }

  const every = boundedInterval()
  return intervalUnit.value === 'minutes'
    ? [`*/${every} * * * *`]
    : [`0 */${every} * * *`]
})

const validationError = computed(() => {
  if (!selectedScriptId.value) {
    return allowTargetSelect.value ? '请选择脚本或工作流' : '请先保存脚本或工作流'
  }
  if (scheduleMode.value === 'custom' && !normalizeCron(customCron.value)) {
    return '请输入 cron 表达式'
  }
  if (presetType.value === 'weekly' && selectedDays.value.length === 0) {
    return '请选择星期'
  }
  if (presetType.value === 'daily_multi' && uniqueTimes(multiTimes.value).length === 0) {
    return '请至少保留一个时刻'
  }
  if (generatedCrons.value.some(expr => expr.split(' ').length !== 5)) {
    return 'cron 表达式需为 5 段'
  }
  return ''
})

const displayError = computed(() => actionError.value || validationError.value)
const displayWarning = computed(() => {
  if (!enabled.value) return ''
  const matches = generatedCrons.value.filter(expr => {
    const normalized = normalizeCron(expr)
    return props.existingSchedules.some(item => {
      const id = item.scheduleId || item.id || 0
      return id !== scheduleId.value &&
        item.enabled &&
        item.scriptId === selectedScriptId.value &&
        normalizeCron(item.cronExpr) === normalized
    })
  })
  if (!matches.length) return ''
  return `已有相同启用规则：${matches.join('，')}，保存后可能重复触发`
})

onMounted(async () => {
  applySchedule(props.schedule)
  if (props.allowTargetSelect) {
    const [scriptList, workflowList] = await Promise.all([
      GetScripts(),
      GetWorkflows(),
    ])
    scripts.value = scriptList || []
    workflows.value = workflowList || []
  }
})

watch(() => props.schedule, applySchedule)
watch(() => props.scriptId, value => {
  if (!props.schedule && !props.allowTargetSelect) selectedScriptId.value = value || 0
})

function cronForTime(value, dayExpr = '*') {
  const time = parseTime(value)
  if (!time) return ''
  return `${time.minute} ${time.hour} * * ${dayExpr}`
}

function parseTime(value) {
  const match = /^(\d{1,2}):(\d{2})$/.exec(value || '')
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null
  return { hour: String(hour), minute: String(minute) }
}

function normalizeCron(value) {
  return (value || '').trim().replace(/\s+/g, ' ')
}

function uniqueTimes(times) {
  return [...new Set(times.filter(time => parseTime(time)))].sort()
}

function sortedDays(days) {
  const order = [1, 2, 3, 4, 5, 6, 0]
  return [...new Set(days)].sort((a, b) => order.indexOf(a) - order.indexOf(b))
}

function boundedInterval() {
  const raw = Number(intervalEvery.value) || 1
  return Math.min(intervalLimit.value, Math.max(1, Math.floor(raw)))
}

function applySchedule(schedule) {
  actionError.value = ''
  toast.value = ''
  selectedScriptId.value = schedule?.scriptId ?? props.scriptId ?? 0
  if (!schedule) return

  enabled.value = schedule.enabled === true || schedule.enabled === 1
  const expr = normalizeCron(schedule.cronExpr)
  const parsed = parseCronToPreset(expr)
  if (!parsed) {
    scheduleMode.value = 'custom'
    customCron.value = expr
    return
  }

  scheduleMode.value = 'preset'
  presetType.value = parsed.type
  if (parsed.time) timeVal.value = parsed.time
  if (parsed.times) multiTimes.value = parsed.times
  if (parsed.days) selectedDays.value = parsed.days
  if (parsed.intervalEvery) intervalEvery.value = parsed.intervalEvery
  if (parsed.intervalUnit) intervalUnit.value = parsed.intervalUnit
}

function parseCronToPreset(expr) {
  const parts = normalizeCron(expr).split(' ')
  if (parts.length !== 5) return null
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  if (dayOfMonth !== '*' || month !== '*') return null

  const minuteInterval = minute.match(/^\*\/(\d+)$/)
  if (minuteInterval && hour === '*' && dayOfWeek === '*') {
    return { type: 'interval', intervalEvery: Number(minuteInterval[1]), intervalUnit: 'minutes' }
  }

  const hourInterval = hour.match(/^\*\/(\d+)$/)
  if (minute === '0' && hourInterval && dayOfWeek === '*') {
    return { type: 'interval', intervalEvery: Number(hourInterval[1]), intervalUnit: 'hours' }
  }

  if (!isNumberField(minute)) return null

  if (/^\d{1,2}(,\d{1,2})+$/.test(hour) && dayOfWeek === '*') {
    const times = hour.split(',').map(h => formatTime(h, minute)).filter(Boolean)
    return times.length ? { type: 'daily_multi', times } : null
  }

  if (!isNumberField(hour)) return null
  const time = formatTime(hour, minute)
  if (!time) return null

  if (dayOfWeek === '*') return { type: 'daily_once', time }
  if (dayOfWeek === '1-5') return { type: 'workday', time }

  const days = parseDayList(dayOfWeek)
  if (days.length) return { type: 'weekly', time, days }
  return null
}

function isNumberField(value) {
  return /^\d{1,2}$/.test(value)
}

function formatTime(hourValue, minuteValue) {
  const hour = Number(hourValue)
  const minute = Number(minuteValue)
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return ''
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`
}

function parseDayList(value) {
  if (!/^([0-7])(?:,[0-7])*$/.test(value)) return []
  return sortedDays(value.split(',').map(v => Number(v) === 7 ? 0 : Number(v)))
}

function addTime() {
  actionError.value = ''
  const last = multiTimes.value[multiTimes.value.length - 1] || '08:00'
  multiTimes.value.push(last)
}

function removeTime(index) {
  actionError.value = ''
  if (multiTimes.value.length === 1) return
  multiTimes.value.splice(index, 1)
}

function formatError(err) {
  const message = err?.message || String(err || '保存失败')
  return message.replace(/^Error:\s*/, '')
}

async function handleSave() {
  actionError.value = ''
  if (validationError.value) return

  saving.value = true
  try {
    for (const cronExpr of generatedCrons.value) {
      const isFirst = cronExpr === generatedCrons.value[0]
      await SaveSchedule({
        id: isEditing.value && isFirst ? scheduleId.value : 0,
        scriptId: selectedScriptId.value,
        cronExpr,
        enabled: enabled.value ? 1 : 0,
      })
    }
    if (isEditing.value && generatedCrons.value.length > 1) {
      toast.value = `已更新 1 条并新增 ${generatedCrons.value.length - 1} 条定时规则`
    } else if (isEditing.value) {
      toast.value = '定时规则已更新'
    } else {
      toast.value = generatedCrons.value.length > 1 ? `已保存 ${generatedCrons.value.length} 条定时规则` : '定时设置成功'
    }
    emit('saved')
    setTimeout(() => emit('close'), 900)
  } catch (err) {
    actionError.value = formatError(err)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(1, 4, 9, .72);
}

.schedule-modal {
  width: min(680px, calc(100vw - 48px));
  max-height: calc(100vh - 48px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  text-align: left;
  background: var(--sidebar-bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, .38);
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px 16px;
  border-bottom: 1px solid var(--border);
}

.eyebrow {
  margin-bottom: 4px;
  color: var(--orange);
  font-size: 12px;
  font-weight: 600;
}

h3 {
  color: var(--text);
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
}

.icon-btn {
  width: 30px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--text-muted);
  font-size: 20px;
  line-height: 1;
}

.icon-btn:hover {
  color: var(--text);
  border-color: var(--text-muted);
}

.modal-body {
  padding: 20px 24px;
  overflow-y: auto;
}

.field {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}

.field-top {
  align-items: flex-start;
}

.field.compact {
  margin-bottom: 0;
}

.field > label {
  color: var(--text-dim);
  font-size: 13px;
  text-align: right;
}

select,
input[type="time"],
.cron-input,
.number-input {
  width: 100%;
  height: 34px;
  padding: 6px 10px;
  background: var(--input-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
}

select:focus,
input[type="time"]:focus,
.cron-input:focus,
.number-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-dim);
}

.segmented {
  display: inline-flex;
  width: fit-content;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--input-bg);
}

.segmented button {
  min-width: 86px;
  height: 32px;
  padding: 0 14px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 14px;
}

.segmented button.active {
  background: var(--accent);
  color: #fff;
}

.weekday-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 8px;
}

.weekday-grid label {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--input-bg);
  color: var(--text-dim);
  font-size: 13px;
}

.weekday-grid label.checked {
  color: var(--text);
  border-color: rgba(47, 129, 247, .55);
  background: var(--accent-dim);
}

.weekday-grid input {
  width: 13px;
  height: 13px;
}

.time-stack {
  display: grid;
  gap: 8px;
  max-width: 220px;
}

.time-line,
.inline-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ghost-btn,
.preset-chips button {
  height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--text-dim);
  font-size: 13px;
}

.ghost-btn:hover,
.preset-chips button:hover {
  color: var(--text);
  border-color: var(--text-muted);
}

.ghost-btn.danger {
  width: 34px;
  padding: 0;
  font-size: 18px;
}

.ghost-btn.danger:hover {
  color: var(--red);
  border-color: rgba(248, 81, 73, .5);
}

.ghost-btn:disabled {
  opacity: .45;
  cursor: not-allowed;
}

.add-time {
  width: fit-content;
}

.number-input {
  max-width: 88px;
}

.inline-controls select {
  max-width: 100px;
}

.preset-chips {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.preview-block {
  margin: 18px 0 16px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
}

.preview-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 600;
}

.preview-title strong {
  color: var(--orange);
  font-size: 12px;
}

.cron-list {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

code {
  padding: 5px 9px;
  border: 1px solid rgba(63, 185, 80, .28);
  border-radius: var(--radius-sm);
  background: var(--green-dim);
  color: var(--green);
  font-family: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
}

.empty-preview {
  color: var(--text-muted);
}

.switch {
  position: relative;
  width: 42px;
  height: 24px;
  display: inline-block;
}

.switch input {
  position: absolute;
  opacity: 0;
}

.switch span {
  position: absolute;
  inset: 0;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface2);
  transition: background .12s, border-color .12s;
}

.switch span::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--text-muted);
  transition: transform .12s, background .12s;
}

.switch input:checked + span {
  border-color: rgba(63, 185, 80, .55);
  background: var(--green-dim);
}

.switch input:checked + span::after {
  transform: translateX(18px);
  background: var(--green);
}

.message {
  margin-top: 14px;
  padding: 9px 12px;
  border-radius: var(--radius);
  font-size: 13px;
}

.message.error {
  color: var(--red);
  background: var(--red-dim);
  border: 1px solid rgba(248, 81, 73, .28);
}

.message.warning {
  color: var(--orange);
  background: var(--orange-dim);
  border: 1px solid rgba(210, 153, 34, .28);
}

.message.success {
  color: var(--green);
  background: var(--green-dim);
  border: 1px solid rgba(63, 185, 80, .28);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px 22px;
  border-top: 1px solid var(--border);
}

.btn-primary,
.btn-cancel {
  min-width: 82px;
  height: 34px;
  border-radius: var(--radius);
  font-size: 14px;
  font-weight: 500;
}

.btn-primary {
  border: 1px solid var(--accent);
  background: var(--accent);
  color: #fff;
}

.btn-primary:hover {
  background: var(--accent-hover);
  border-color: var(--accent-hover);
}

.btn-primary:disabled {
  opacity: .55;
  cursor: not-allowed;
}

.btn-cancel {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-dim);
}

.btn-cancel:hover {
  color: var(--text);
  border-color: var(--text-muted);
}

@media (max-width: 720px) {
  .schedule-modal {
    width: 100%;
  }

  .field {
    grid-template-columns: 1fr;
    gap: 7px;
  }

  .field > label {
    text-align: left;
  }

  .weekday-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
