import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

export const useMainStore = defineStore('main', () => {
  const selectedScriptID = ref(null)
  const currentView = ref('script')
  const runningScripts = reactive(new Set())
  const currentLogs = ref([])
  const scriptListVersion = ref(0)
  const selectedScriptWorkDir = ref('')

  function setScript(id) {
    selectedScriptID.value = id
    currentView.value = 'script'
    currentLogs.value = []
    selectedScriptWorkDir.value = ''
  }

  function setView(view) { currentView.value = view }
  function addLog(entry) { currentLogs.value.push(entry) }
  function clearLogs() { currentLogs.value = [] }
  function setRunning(scriptID, running) {
    if (running) runningScripts.add(scriptID)
    else runningScripts.delete(scriptID)
  }
  function refreshScriptList() { scriptListVersion.value++ }

  return { selectedScriptID, currentView, runningScripts, currentLogs, scriptListVersion, selectedScriptWorkDir, setScript, setView, addLog, clearLogs, setRunning, refreshScriptList }
})
