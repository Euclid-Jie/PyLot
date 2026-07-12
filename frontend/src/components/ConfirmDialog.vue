<template>
  <div class="dialog-overlay" @click.self="$emit('cancel')" @keydown.esc="$emit('cancel')">
    <section ref="dialogEl" class="confirm-dialog" role="alertdialog" aria-modal="true" :aria-labelledby="titleId" tabindex="-1">
      <header>
        <div>
          <span>请确认操作</span>
          <h3 :id="titleId">{{ title }}</h3>
        </div>
        <button class="ui-icon-btn" aria-label="关闭" title="关闭" @click="$emit('cancel')"><UiIcon name="x" /></button>
      </header>
      <p>{{ message }}</p>
      <footer>
        <button class="ui-btn" @click="$emit('cancel')">取消</button>
        <button class="ui-btn danger solid" @click="$emit('confirm')">{{ confirmText }}</button>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { nextTick, onMounted, ref } from 'vue'
import UiIcon from './UiIcon.vue'

defineProps({
  title: { type: String, default: '确认删除？' },
  message: { type: String, default: '此操作无法撤销。' },
  confirmText: { type: String, default: '删除' },
})
defineEmits(['confirm', 'cancel'])

const dialogEl = ref(null)
const titleId = `dialog-title-${Math.random().toString(36).slice(2)}`
onMounted(() => nextTick(() => dialogEl.value?.focus()))
</script>

<style scoped>
.confirm-dialog { width: min(440px, calc(100vw - 40px)); border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--surface-raised); box-shadow: var(--shadow-dialog); outline: none; }
header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 20px 20px 12px; }
header span { display: block; margin-bottom: 3px; color: var(--red); font-size: 12px; font-weight: 600; }
h3 { color: var(--text); font-size: 17px; font-weight: 600; }
p { padding: 0 20px 20px; color: var(--text-dim); font-size: 13px; line-height: 1.7; }
footer { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--border); background: var(--surface); }
</style>
