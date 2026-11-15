import { useEffect, useState } from 'react'
import { api } from '../../api'
import type { AppEvent } from '../../types'

export default function Events() {
  const [events, setEvents] = useState<AppEvent[]>([])
  const [filter, setFilter] = useState('')
  const [levelFilter, setLevelFilter] = useState<string>('')

  useEffect(() => {
    const es = new EventSource('/api/stream/events')
    es.addEventListener('event', (e: MessageEvent) => {
      try {
        const batch: AppEvent[] = JSON.parse(e.data)
        setEvents((prev) => [...batch, ...prev].slice(0, 1000))
      } catch {}
    })
    return () => es.close()
  }, [])

  const filtered = events.filter((e) => {
    if (levelFilter && e.level !== levelFilter) return false
    if (filter && !JSON.stringify(e).toLowerCase().includes(filter.toLowerCase())) return false
    return true
  })

  return (
    <div>
      <h1>Live events</h1>
      <div className="form-row">
        <input
          placeholder="Filter by text..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          style={{ flex: 1 }}
        />
        <select value={levelFilter} onChange={(e) => setLevelFilter(e.target.value)}>
          <option value="">All levels</option>
          <option value="Debug">Debug</option>
          <option value="Info">Info</option>
          <option value="Warning">Warning</option>
          <option value="Error">Error</option>
        </select>
      </div>

      <ul className="event-list big">
        {filtered.map((ev, i) => (
          <li key={i} className={`event level-${ev.level.toLowerCase()}`}>
            <span className="event-time">{new Date(ev.time).toLocaleString()}</span>
            <span className={`pill level-${ev.level.toLowerCase()}`}>{ev.level}</span>
            <span className="event-comp">{ev.component}</span>
            <span className="event-msg">
              {ev.message}
              {ev.job_id > 0 && ` · job #${ev.job_id}`}
              {ev.worker_id > 0 && ` · worker #${ev.worker_id}`}
            </span>
          </li>
        ))}
        {filtered.length === 0 && <li className="muted">Waiting for events...</li>}
      </ul>
    </div>
  )
}
