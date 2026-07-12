<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <section class="args-dialog" role="dialog" aria-modal="true" aria-labelledby="args-title">
      <header><div><span>运行配置</span><h3 id="args-title">临时参数</h3></div><button class="ui-icon-btn" title="关闭" aria-label="关闭" @click="$emit('close')"><UiIcon name="x" /></button></header>
      <div class="dialog-body">
        <label>当前固定参数</label><code>{{ fixedArgs || '无' }}</code>
        <label for="temp-args">本次运行参数</label><input id="temp-args" v-model="tempArgs" class="ui-input ui-mono" placeholder="输入参数；留空则使用固定参数" @keydown.enter="$emit('run', tempArgs)" />
        <p>临时参数仅对本次运行生效，不会修改脚本配置。</p>
      </div>
      <footer><button class="ui-btn" @click="$emit('run', '')">使用固定参数</button><button class="ui-btn primary" @click="$emit('run', tempArgs)"><UiIcon name="play" />运行</button></footer>
    </section>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import UiIcon from './UiIcon.vue'
defineProps({ fixedArgs: String })
defineEmits(['run', 'close'])
const tempArgs = ref('')
</script>

<style scoped>
.args-dialog { width: min(520px, calc(100vw - 40px)); overflow: hidden; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--surface-raised); box-shadow: var(--shadow-dialog); }
header { display: flex; align-items: flex-start; justify-content: space-between; padding: 18px 20px 14px; border-bottom: 1px solid var(--border); }
header span { color: var(--text-muted); font-size: 12px; } h3 { font-size: 17px; font-weight: 600; }
.dialog-body { display: grid; gap: 8px; padding: 18px 20px; }
label { color: var(--text-dim); font-size: 12px; font-weight: 600; }
code { display: block; overflow-x: auto; margin-bottom: 10px; padding: 8px 10px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--input-bg); color: var(--green); font-family: var(--mono); font-size: 12px; white-space: nowrap; }
p { color: var(--text-muted); font-size: 12px; }
footer { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--border); background: var(--surface); }
</style>
