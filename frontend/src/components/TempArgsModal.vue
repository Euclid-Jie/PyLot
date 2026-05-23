<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>临时参数</h3>
      <p class="hint">当前固定参数：<code>{{ fixedArgs || '（无）' }}</code></p>
      <label>临时参数（将完全替换固定参数）：</label>
      <input v-model="tempArgs" placeholder="输入临时参数..." />
      <div class="modal-actions">
        <button class="btn-primary" @click="$emit('run', tempArgs)">使用临时参数运行</button>
        <button class="btn-secondary" @click="$emit('run', '')">使用固定参数运行</button>
        <button class="btn-cancel" @click="$emit('close')">取消</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
defineProps({ fixedArgs: String })
defineEmits(['run', 'close'])
const tempArgs = ref('')
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { background: #16213e; border: 1px solid #0f3460; border-radius: 8px; padding: 24px; min-width: 400px; }
h3 { margin-bottom: 12px; }
.hint { font-size: 13px; color: #aaa; margin-bottom: 12px; }
code { background: #252526; padding: 2px 6px; border-radius: 3px; }
label { font-size: 13px; color: #aaa; display: block; margin-bottom: 6px; }
input { width: 100%; padding: 6px 10px; background: #252526; border: 1px solid #444; color: #e0e0e0; border-radius: 4px; font-size: 13px; margin-bottom: 16px; }
.modal-actions { display: flex; gap: 8px; }
.btn-primary { background: #4caf50; color: #fff; border: none; padding: 7px 14px; border-radius: 4px; }
.btn-secondary { background: #2196f3; color: #fff; border: none; padding: 7px 14px; border-radius: 4px; }
.btn-cancel { background: #555; color: #ccc; border: none; padding: 7px 14px; border-radius: 4px; }
</style>
