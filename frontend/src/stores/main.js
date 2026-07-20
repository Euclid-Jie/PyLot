import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { GetLatestLog } from '../../wailsjs/go/main/App.js'

export const useMainStore = defineStore('main', () => {
  const selectedScriptID = ref(null)
  const savedScriptList = localStorage.getItem('selectedScriptList') || 'all'
  const selectedScriptListId = ref(['all', 'unassigned'].includes(savedScriptList) ? savedScriptList : Number(savedScriptList))
  const currentView = ref('script')
  const runningScripts = reactive(new Set())
  const currentLogs = ref([])
  const scriptListVersion = ref(0)
  const selectedScriptWorkDir = ref('')
  const selectedWorkflowId = ref(null)
  const newWorkflowTick = ref(0)
  const isDirty = ref(false)
  const navigationBlocked = ref(false)
  let pendingNavigation = null

  function navigate(action) {
    if (isDirty.value) {
      pendingNavigation = action
      navigationBlocked.value = true
      return false
    }
    action()
    return true
  }

  function markDirty() { isDirty.value = true }
  function clearDirty() { isDirty.value = false }
  function confirmNavigation() {
    const action = pendingNavigation
    pendingNavigation = null
    navigationBlocked.value = false
    isDirty.value = false
    action?.()
  }
  function cancelNavigation() {
    pendingNavigation = null
    navigationBlocked.value = false
  }

  function setScript(id) {
    if (currentView.value === 'script' && selectedScriptID.value === id) return true
    return navigate(() => {
      selectedScriptID.value = id
      currentView.value = 'script'
      currentLogs.value = []
      selectedScriptWorkDir.value = ''
    })
  }

  function setScriptList(id) {
    if (currentView.value === 'script' && selectedScriptListId.value === id && selectedScriptID.value === null) return true
    return navigate(() => {
      selectedScriptListId.value = id
      selectedScriptID.value = null
      currentView.value = 'script'
      currentLogs.value = []
      selectedScriptWorkDir.value = ''
      localStorage.setItem('selectedScriptList', String(id))
    })
  }

  function showScriptInList(id) {
    selectedScriptListId.value = id > 0 ? id : 'unassigned'
    localStorage.setItem('selectedScriptList', String(selectedScriptListId.value))
  }

  function closeScriptDetail() {
    return navigate(() => {
      selectedScriptID.value = null
      currentLogs.value = []
      selectedScriptWorkDir.value = ''
    })
  }

  function setScriptFromWorkflow(id) {
    if (currentView.value === 'script' && selectedScriptID.value === id) return true
    return navigate(async () => {
      selectedScriptID.value = id
      currentView.value = 'script'
      selectedScriptListId.value = 'all'
      localStorage.setItem('selectedScriptList', 'all')
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
    })
  }

  function setView(view) {
    if (currentView.value === view) return true
    return navigate(() => { currentView.value = view })
  }
  function openWorkflow(id) {
    if (currentView.value === 'workflow' && selectedWorkflowId.value === id) return true
    return navigate(() => { selectedWorkflowId.value = id; selectedScriptID.value = null; currentView.value = 'workflow' })
  }
  function newWorkflow() { return navigate(() => { selectedWorkflowId.value = null; newWorkflowTick.value++; currentView.value = 'workflow' }) }
  function addLog(entry) { currentLogs.value.push(entry) }
  function clearLogs() { currentLogs.value = [] }
  function setRunning(scriptID, running) {
    if (running) runningScripts.add(scriptID)
    else runningScripts.delete(scriptID)
  }
  function refreshScriptList() { scriptListVersion.value++ }

  return { selectedScriptID, selectedScriptListId, currentView, runningScripts, currentLogs, scriptListVersion, selectedScriptWorkDir, selectedWorkflowId, newWorkflowTick, isDirty, navigationBlocked, setScript, setScriptList, showScriptInList, closeScriptDetail, setScriptFromWorkflow, setView, openWorkflow, newWorkflow, addLog, clearLogs, setRunning, refreshScriptList, markDirty, clearDirty, confirmNavigation, cancelNavigation, requestNavigation: navigate }
})
