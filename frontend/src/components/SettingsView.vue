<template>
  <div class="settings-page">
    <h2>设置</h2>

    <div class="section">
      <div class="section-title">外观</div>
      <div class="setting-row">
        <label>主题</label>
        <div class="toggle-group">
          <button :class="{ active: theme === 'dark' }" @click="setTheme('dark')">深色</button>
          <button :class="{ active: theme === 'light' }" @click="setTheme('light')">浅色</button>
        </div>
      </div>
      <div class="setting-row">
        <label>字体</label>
        <select v-model="font" @change="setFont(font)">
          <option value="system-ui, sans-serif">系统默认</option>
          <option value="'Microsoft YaHei', sans-serif">微软雅黑</option>
          <option value="'Segoe UI', sans-serif">Segoe UI</option>
          <option value="'JetBrains Mono', monospace">JetBrains Mono</option>
          <option value="'Consolas', monospace">Consolas</option>
        </select>
      </div>
    </div>

    <div class="section">
      <div class="section-title">环境</div>
      <div class="setting-row">
        <label>全局 .env 文件</label>
        <div class="path-row">
          <input v-model="envPath" placeholder="选择 .env 文件路径" />
          <button @click="browseEnv">浏览</button>
        </div>
      </div>
      <button class="btn-save" @click="saveEnv">保存</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GetGlobalConfig, SaveGlobalConfig, OpenFileDialog } from '../../wailsjs/go/main/App.js'

const envPath = ref('')
const theme = ref(localStorage.getItem('theme') || 'dark')
const font = ref(localStorage.getItem('font') || 'system-ui, sans-serif')

onMounted(async () => {
  const cfg = await GetGlobalConfig()
  envPath.value = cfg.envFilePath || ''
})

function setTheme(t) {
  theme.value = t
  localStorage.setItem('theme', t)
  document.documentElement.setAttribute('data-theme', t)
}

function setFont(f) {
  localStorage.setItem('font', f)
  document.documentElement.style.setProperty('--font', f)
}

async function browseEnv() {
  const p = await OpenFileDialog('选择 .env 文件')
  if (p) envPath.value = p
}

async function saveEnv() {
  await SaveGlobalConfig({ envFilePath: envPath.value })
}
</script>

<style scoped>
.settings-page { padding: 8px; max-width: 560px; }
h2 { font-size: 15px; font-weight: 600; margin-bottom: 20px; color: var(--text); }
.section { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin-bottom: 16px; }
.section-title { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: .05em; margin-bottom: 14px; }
.setting-row { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.setting-row label { width: 110px; font-size: 13px; color: var(--text-muted); flex-shrink: 0; }
.toggle-group { display: flex; border: 1px solid var(--border); border-radius: 4px; overflow: hidden; }
.toggle-group button { padding: 5px 16px; background: none; border: none; color: var(--text-muted); font-size: 13px; }
.toggle-group button.active { background: var(--accent); color: #fff; }
select { padding: 5px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: 13px; }
.path-row { display: flex; gap: 6px; flex: 1; }
.path-row input { flex: 1; padding: 5px 8px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: 4px; font-size: 13px; }
.path-row button { padding: 5px 10px; background: var(--surface2); color: var(--text-muted); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; }
.btn-save { padding: 6px 16px; background: var(--accent); color: #fff; border: none; border-radius: 4px; font-size: 13px; margin-top: 4px; }
</style>
