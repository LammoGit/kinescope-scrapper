import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../../api'
import type { Job, Metrics, AppEvent } from '../../types'
import ProgressBar from '../../components/ProgressBar'

export default function JobDetail() {
  const { jobId } = useParams()
  const id = Number(jobId)

  const [job, setJob] = useState<Job | null>(null)
  const [items, setItems] = useState<number[]>([])
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [events, setEvents] = useState<AppEvent[]>([])
  const [err, setErr] = useState<string | null>(null)

  const refresh = () => {
    api.getJob(id).then(setJob).catch((e) => setErr(e.message))
    api.getJobItems(id).then(setItems).catch(() => {})
    api.getJobEvents(id).then((evs) => setEvents((evs || []).slice(-50).reverse())).catch(() => {})
  }
  useEffect(refresh, [id])

  // Live events
  useEffect(() => {
    const es = new EventSource(`/api/stream/events?jobs=${id}`)
    es.addEventListener('event', (e: MessageEvent) => {
      try {
        const batch: AppEvent[] = JSON.parse(e.data)
        setEvents((prev) => [...batch, ...prev].slice(0, 200))
      } catch {}
    })
    return () => es.close()
  }, [id])

  // Live metrics
  useEffect(() => {
    const es = new EventSource(`/api/stream/jobs/metrics?jobs=${id}`)
    es.addEventListener('job.metrics', (e: MessageEvent) => {
      try {
        const arr: Metrics[] = JSON.parse(e.data)
        if (arr && arr.length) setMetrics(arr[0])
      } catch {}
    })
    return () => es.close()
  }, [id])

  const action = async (fn: () => Promise<any>) => {
    try { await fn(); refresh() } catch (e: any) { setErr(e.message) }
  }

  if (!job) return <div className="loading">{err || 'Loading...'}</div>

  const total = Math.max(0, job.config.range.to - job.config.range.from + 1)

  return (
    <div>
      <div className="page-header">
        <h1>Job #{job.id} · {job.config.name}</h1>
        <Link to="/admin/jobs">← All jobs</Link>
      </div>

      {err && <div className="error">{err}</div>}

      <div className="stat-grid">
        <div className="stat"><div className="stat-value">{job.status}</div><div className="stat-label">Status</div></div>
        <div className="stat"><div className="stat-value">{items.length}</div><div className="stat-label">Downloaded</div></div>
        <div className="stat"><div className="stat-value">{job.workers?.length ?? 0}</div><div className="stat-label">Assigned workers</div></div>
        <div className="stat"><div className="stat-value">{metrics?.identifiers_per_second?.toFixed(1) ?? '—'}</div><div className="stat-label">IDs / sec</div></div>
        <div className="stat"><div className="stat-value">{metrics?.active_downloads ?? '—'}</div><div className="stat-label">Active downloads</div></div>
        <div className="stat"><div className="stat-value">{metrics?.final_errors ?? '—'}</div><div className="stat-label">Errors</div></div>
      </div>

      <div className="card">
        <h2>Progress</h2>
        <ProgressBar value={items.length} max={total} label="Downloaded videos" />
      </div>

      <div className="card">
        <h2>Controls</h2>
        <div className="row-actions">
          <button onClick={() => action(() => api.startJob(job.id))}>Start</button>
          <button onClick={() => action(() => api.pauseJob(job.id))}>Pause</button>
          <button onClick={() => action(() => api.resumeJob(job.id))}>Resume</button>
          <button onClick={() => action(() => api.stopJob(job.id))}>Stop</button>
          <button onClick={() => { if (confirm('Delete job?')) action(() => api.deleteJob(job.id)) }}>Delete</button>
        </div>
      </div>

      <div className="card">
        <h2>Configuration</h2>
        <pre className="code">{JSON.stringify(job.config, null, 2)}</pre>
      </div>

      <div className="card">
        <h2>Downloaded videos ({items.length})</h2>
        <div className="chip-grid">
          {items.slice(0, 200).map((vid) => (
            <Link key={vid} to={`/video/${vid}`} className="chip">#{vid}</Link>
          ))}
          {items.length > 200 && <span className="muted">…and {items.length - 200} more</span>}
          {items.length === 0 && <span className="muted">No videos downloaded yet.</span>}
        </div>
      </div>

      <div className="card">
        <h2>Live events ({events.length})</h2>
        <ul className="event-list">
          {events.slice(0, 50).map((ev, i) => (
            <li key={i} className={`event level-${ev.level.toLowerCase()}`}>
              <span className="event-time">{new Date(ev.time).toLocaleTimeString()}</span>
              <span className={`pill level-${ev.level.toLowerCase()}`}>{ev.level}</span>
              <span className="event-comp">{ev.component}</span>
              <span className="event-msg">{ev.message}</span>
            </li>
          ))}
          {events.length === 0 && <li className="muted">No events yet.</li>}
        </ul>
      </div>
    </div>
  )
}
