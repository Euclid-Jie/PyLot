export const DEFAULT_FONT = '"Segoe UI Variable", "Segoe UI", "Microsoft YaHei UI", sans-serif'

export function resolveFont() {
  const stored = localStorage.getItem('font')
  if (!stored) return DEFAULT_FONT
  if (stored === 'system-ui, sans-serif') {
    localStorage.setItem('font', DEFAULT_FONT)
    return DEFAULT_FONT
  }
  return stored
}
