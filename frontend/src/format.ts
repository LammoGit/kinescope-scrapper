// Shared formatting helpers for the admin pages

/** A range with to = 0 is open-ended */
export function isOpenRange(to: number): boolean {
  return !to || to <= 0
}

/** Display a range, open-ended ranges get an infinity sign */
export function fmtRange(from: number, to: number): string {
  return isOpenRange(to) ? `${from} – ∞` : `${from} – ${to}`
}

/** Number of identifiers the range covers, null when open-ended */
export function rangeSize(from: number, to: number): number | null {
  if (isOpenRange(to)) return null
  return Math.max(0, to - from + 1)
}

/** Speeds in identifiers per second */
export function fmtIds(n: number | undefined): string {
  if (n == null || !isFinite(n)) return '—'
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k/s`
  return `${n.toFixed(1)}/s`
}

/** Speeds and sizes in bytes per second */
export function fmtBytesPerSec(n: number | undefined): string {
  if (n == null || !isFinite(n)) return '—'
  return `${fmtBytes(n)}/s`
}

export function fmtBytes(n: number | undefined): string {
  if (n == null || !isFinite(n)) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v >= 100 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`
}

/** Agent uptime given in seconds */
export function fmtUptime(seconds: number | undefined): string {
  if (seconds == null || seconds < 0) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

/** Remaining time for a finite range, null when the speed is zero or unknown */
export function fmtEta(remaining: number | null, perSecond: number): string {
  if (remaining == null) return '∞'
  if (!perSecond || perSecond <= 0) return '—'
  const s = Math.ceil(remaining / perSecond)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) return `~${h}h ${m}m`
  if (m > 0) return `~${m}m ${sec}s`
  return `~${sec}s`
}

/**
 * Parse a range in the formats from the lab spec:
 * `[A,B]` inclusive bounds, `A` shorthand for `[0,A]`, `[A,]` and empty for
 * open-ended. Returns null when the input is invalid.
 */
export function parseRange(text: string): { from: number; to: number } | null {
  const s = text.trim()
  if (s === '') return { from: 0, to: 0 }
  const bracket = s.match(/^\[\s*(\d*)\s*,\s*(\d*)\s*\]$/)
  if (bracket) {
    const from = bracket[1] === '' ? 0 : Number(bracket[1])
    const to = bracket[2] === '' ? 0 : Number(bracket[2])
    if (to > 0 && from > to) return null
    return { from, to }
  }
  if (/^\d+$/.test(s)) return { from: 0, to: Number(s) }
  return null
}
