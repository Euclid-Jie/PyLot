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
      <button class="btn-save" @click="saveConfig">保存</button>
    </div>

    <div class="section">
      <div class="section-title">飞书通知</div>
      <div class="setting-row">
        <label>lark-cli 路径</label>
        <input class="full-input" v-model="larkCLI" placeholder="如 C:\...\lark-cli.cmd" />
      </div>
      <div class="setting-row">
        <label>Open ID</label>
        <input class="full-input" v-model="larkOpenID" placeholder="ou_xxxxxxxx" />
      </div>
      <button class="btn-save" @click="saveConfig">保存</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GetGlobalConfig, SaveGlobalConfig, OpenFileDialog } from '../../wailsjs/go/main/App.js'

const envPath = ref('')
const larkCLI = ref('')
const larkOpenID = ref('')
const theme = ref(localStorage.getItem('theme') || 'dark')
const font = ref(localStorage.getItem('font') || 'system-ui, sans-serif')

onMounted(async () => {
  const cfg = await GetGlobalConfig()
  envPath.value = cfg.envFilePath || ''
  larkCLI.value = cfg.larkCliPath || ''
  larkOpenID.value = cfg.larkOpenId || ''
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

async function saveConfig() {
  await SaveGlobalConfig({ envFilePath: envPath.value, larkCliPath: larkCLI.value, larkOpenId: larkOpenID.value })
}
</script>

<style scoped>
.settings-page { padding: 0; max-width: 540px; }
h2 { font-size: 20px; font-weight: 600; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--border); color: var(--text); }
.section { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); padding: 20px; margin-bottom: 16px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--text-muted); text-transform: uppercase; letter-spacing: .04em; margin-bottom: 16px; }
.setting-row { display: flex; align-items: center; gap: 14px; margin-bottom: 14px; }
.setting-row:last-child { margin-bottom: 0; }
.setting-row label { width: 110px; font-size: 14px; color: var(--text-dim); flex-shrink: 0; }
.toggle-group { display: flex; border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.toggle-group button { padding: 6px 18px; background: transparent; border: none; color: var(--text-muted); font-size: 14px; font-weight: 500; transition: background .12s, color .12s; }
.toggle-group button.active { background: var(--accent); color: #fff; }
.toggle-group button:hover:not(.active) { background: var(--surface2); color: var(--text); }
select { padding: 7px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: 14px; transition: border-color .12s; }
select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.path-row { display: flex; gap: 8px; flex: 1; }
.path-row input { flex: 1; padding: 7px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: 14px; transition: border-color .12s; }
.path-row input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.path-row button { padding: 6px 12px; background: var(--surface2); color: var(--text-dim); border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; font-weight: 500; transition: background .12s, color .12s; }
.path-row button:hover { background: var(--surface); color: var(--text); border-color: var(--text-muted); }
.btn-save { padding: 6px 18px; background: var(--accent); color: #fff; border: 1px solid var(--accent); border-radius: var(--radius); font-size: 14px; font-weight: 500; margin-top: 6px; transition: background .12s; }
.btn-save:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.full-input { flex: 1; padding: 7px 10px; background: var(--input-bg); border: 1px solid var(--border); color: var(--text); border-radius: var(--radius); font-size: 14px; transition: border-color .12s; }
.full-input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
</style>
