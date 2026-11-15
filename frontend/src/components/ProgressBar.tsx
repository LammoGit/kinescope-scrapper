export default function ProgressBar({ value, max, label }: { value: number; max: number; label?: string }) {
  const pct = max > 0 ? Math.min(100, (value / max) * 100) : 0
  return (
    <div className="progressbar">
      {label && <div className="progressbar-label">{label}</div>}
      <div className="progressbar-track">
        <div className="progressbar-fill" style={{ width: `${pct}%` }} />
      </div>
      <div className="progressbar-text">{value} / {max}</div>
    </div>
  )
}
