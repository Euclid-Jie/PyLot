import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { GetLatestLog } from '../../wailsjs/go/main/App.js'

export const useMainStore = defineStore('main', () => {
  const selectedScriptID = ref(null)
  const currentView = ref('script')
  const runningScripts = reactive(new Set())
  const currentLogs = ref([])
  const scriptListVersion = ref(0)
  const selectedScriptWorkDir = ref('')
  const selectedWorkflowId = ref(null)
  const newWorkflowTick = ref(0)

  function setScript(id) {
    selectedScriptID.value = id
    currentView.value = 'script'
    currentLogs.value = []
    selectedScriptWorkDir.value = ''
  }

  async function setScriptFromWorkflow(id) {
    selectedScriptID.value = id
    currentView.value = 'script'
    selectedWorkflowId.value = null
    selectedScriptWorkDir.value = ''
    const rec = await GetLatestLog(id)
    if (rec?.logOutput) {
      currentLogs.value = rec.logOutput.split('\n').filter(Boolean).map(line => ({
        scriptID: id, line, isError: rec.isError === 1,
        timestamp: new Date(rec.startedAt).toLocaleTimeString('zh-CN', { hour12: false })
      }))
    } else {
      currentLogs.value = []
    }
  }

  function setView(view) { currentView.value = view }
  function openWorkflow(id) { selectedWorkflowId.value = id; selectedScriptID.value = null; currentView.value = 'workflow' }
  function newWorkflow() { selectedWorkflowId.value = null; newWorkflowTick.value++; currentView.value = 'workflow' }
  function addLog(entry) { currentLogs.value.push(entry) }
  function clearLogs() { currentLogs.value = [] }
  function setRunning(scriptID, running) {
    if (running) runningScripts.add(scriptID)
    else runningScripts.delete(scriptID)
  }
  function refreshScriptList() { scriptListVersion.value++ }

  return { selectedScriptID, currentView, runningScripts, currentLogs, scriptListVersion, selectedScriptWorkDir, selectedWorkflowId, newWorkflowTick, setScript, setScriptFromWorkflow, setView, openWorkflow, newWorkflow, addLog, clearLogs, setRunning, refreshScriptList }
})
