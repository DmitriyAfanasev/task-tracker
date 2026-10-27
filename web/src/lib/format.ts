export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('ru-RU', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}

export function formatDuration(seconds: number | null): string {
  if (seconds == null) return '—'
  if (seconds < 60) return `${Math.round(seconds)} с`
  if (seconds < 3600) return `${Math.round(seconds / 60)} мин`
  if (seconds < 86400) {
    const h = seconds / 3600
    return `${h.toFixed(1)} ч`
  }
  const d = seconds / 86400
  return `${d.toFixed(1)} д`
}

export function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '—'
  if (typeof value === 'string') return value || '—'
  return String(value)
}
