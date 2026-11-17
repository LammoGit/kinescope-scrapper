import { useEffect, useState } from 'react'
import { api, EventStream } from '../../api'
import type { AppEvent } from '../../types'

// Empty value matches anything
type Filters = {
  text: string
  level: string
  result: string
  job: string
  worker: string
  range: string
  resource: string
  from: string
  to: string
}

const emptyFilters: Filters = {
  text: '', level: '', result: '', job: '', worker: '', range: '', resource: '', from: '', to: '',
}

function matches(ev: AppEvent, f: Filters): boolean {
  if (f.level && ev.level !== f.level) return false
  if (f.result && ev.result !== f.result) return false
  if (f.job && ev.job_id !== Number(f.job)) return false
  if (f.worker && ev.worker_id !== Number(f.worker)) return false
  if (f.range && ev.range_id !== Number(f.range)) return false
  if (f.resource && ev.resource_id !== Number(f.resource)) return false
  if (f.text) {
    const hay = `${ev.message} ${ev.component} ${ev.operation_type} ${ev.error_code}`.toLowerCase()
    if (!hay.includes(f.text.toLowerCase())) return false
  }
  const t = new Date(ev.time).getTime()
  if (f.from && t < new Date(f.from).getTime()) return false
  if (f.to && t > new Date(f.to).getTime()) return false
  return true
}

export default function Events() {
  const [events, setEvents] = useState<AppEvent[]>([])
  const [filters, setFilters] = useState<Filters>(emptyFilters)

  useEffect(() => {
    const es = new EventStream('/stream/events')
    es.addEventListener('event', (e) => {
      try {
        const batch = api.asArray<AppEvent>(JSON.parse(e.data))
        setEvents((prev) => [...batch, ...prev].slice(0, 1000))
      } catch {}
    })
    return () => es.close()
  }, [])

  const set = (patch: Partial<Filters>) => setFilters({ ...filters, ...patch })
  const filtered = events.filter((e) => matches(e, filters))

  return (
    <div>
      <div className="page-header">
        <h1>Live events</h1>
        <button onClick={() => setFilters(emptyFilters)}>Reset filters</button>
      </div>

      <div className="filters-grid">
        <label>Text
          <input placeholder="Message, component, operation..." value={filters.text}
            onChange={(e) => set({ text: e.target.value })} />
        </label>
        <label>Level
          <select value={filters.level} onChange={(e) => set({ level: e.target.value })}>
            <option value="">All levels</option>
            <option value="Debug">Debug</option>
            <option value="Info">Info</option>
            <option value="Warning">Warning</option>
            <option value="Error">Error</option>
          </select>
        </label>
        <label>Result
          <select value={filters.result} onChange={(e) => set({ result: e.target.value })}>
            <option value="">All results</option>
            <option value="success">success</option>
            <option value="fail">fail</option>
          </select>
        </label>
        <label>Job ID
          <input type="number" placeholder="any" value={filters.job}
            onChange={(e) => set({ job: e.target.value })} />
        </label>
        <label>Worker ID
          <input type="number" placeholder="any" value={filters.worker}
            onChange={(e) => set({ worker: e.target.value })} />
        </label>
        <label>Range ID
          <input type="number" placeholder="any" value={filters.range}
            onChange={(e) => set({ range: e.target.value })} />
        </label>
        <label>Resource ID
          <input type="number" placeholder="any" value={filters.resource}
            onChange={(e) => set({ resource: e.target.value })} />
        </label>
        <label>From
          <input type="datetime-local" value={filters.from}
            onChange={(e) => set({ from: e.target.value })} />
        </label>
        <label>To
          <input type="datetime-local" value={filters.to}
            onChange={(e) => set({ to: e.target.value })} />
        </label>
      </div>

      <ul className="event-list big">
        {filtered.map((ev, i) => (
          <li key={i} className={`event level-${ev.level.toLowerCase()}`}>
            <span className="event-time">{new Date(ev.time).toLocaleString()}</span>
            <span className={`pill level-${ev.level.toLowerCase()}`}>{ev.level}</span>
            <span className="event-comp">{ev.component}</span>
            <span className={`pill ${ev.result === 'success' ? 'status-completed' : 'status-failed'}`}>{ev.result}</span>
            <span className="event-msg">
              {ev.message}
              {ev.job_id > 0 && ` · job #${ev.job_id}`}
              {ev.worker_id > 0 && ` · worker #${ev.worker_id}`}
              {ev.range_id > 0 && ` · range #${ev.range_id}`}
              {ev.resource_id > 0 && ` · resource #${ev.resource_id}`}
            </span>
          </li>
        ))}
        {filtered.length === 0 && <li className="muted">Waiting for events...</li>}
      </ul>
    </div>
  )
}
