export const STATUS_LABELS = {
  idle: '', running: '⟳ 运行中', success: '✓ 完成', error: '✗ 失败', timeout: '⏱ 超时', killed: '⊘ 已停止'
}
export const statusLabel = s => STATUS_LABELS[s] || ''
