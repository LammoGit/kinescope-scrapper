import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api, EventStream } from '../../api'
import type { Job, Metrics, AppEvent, Worker } from '../../types'
import ProgressBar from '../../components/ProgressBar'
import { fmtIds, fmtBytesPerSec, fmtEta, fmtRange, rangeSize } from '../../format'

export default function JobDetail() {
  const { jobId } = useParams()
  const id = Number(jobId)

  const [job, setJob] = useState<Job | null>(null)
  const [items, setItems] = useState<number[]>([])
  const [metrics, setMetrics] = useState<Metrics | null>(null)
  const [events, setEvents] = useState<AppEvent[]>([])
  const [workers, setWorkers] = useState<Worker[]>([])
  const [itemFilter, setItemFilter] = useState('')
  const [err, setErr] = useState<string | null>(null)

  const refresh = () => {
    api.getJob(id).then(setJob).catch((e) => setErr(e.message))
    api.getJobItems(id).then(setItems).catch(() => {})
    api.getJobEvents(id).then((evs) => setEvents((evs || []).slice(-50).reverse())).catch(() => {})
    api.listWorkers().then(setWorkers).catch(() => {})
  }
  useEffect(refresh, [id])

  // Live events
  useEffect(() => {
    const es = new EventStream(`/stream/events?jobs=${id}`)
    es.addEventListener('event', (e) => {
      try {
        const batch = api.asArray<AppEvent>(JSON.parse(e.data))
        setEvents((prev) => [...batch, ...prev].slice(0, 200))
      } catch {}
    })
    return () => es.close()
  }, [id])

  // Live metrics
  useEffect(() => {
    const es = new EventStream(`/stream/jobs/metrics?jobs=${id}`)
    es.addEventListener('job.metrics', (e) => {
      try {
        const arr = api.asArray<Metrics>(JSON.parse(e.data))
        if (arr.length) setMetrics(arr[arr.length - 1])
      } catch {}
    })
    return () => es.close()
  }, [id])

  // Live job state changes (start/pause/stop coming from elsewhere)
  useEffect(() => {
    const es = new EventStream(`/stream/jobs/updates?jobs=${id}`)
    es.addEventListener('job.state_change', refresh)
    es.addEventListener('job.update', refresh)
    return () => es.close()
  }, [id])

  const action = async (fn: () => Promise<any>) => {
    try { await fn(); refresh() } catch (e: any) { setErr(e.message) }
  }

  // Save the rate limit and request timeout
  const [limits, setLimits] = useState<{ rps: string; timeout: string } | null>(null)
  const limitsValue = limits ?? (job ? {
    rps: String(job.config.global_rate_limit_rps ?? ''),
    timeout: String(job.config.request_timeout_seconds ?? ''),
  } : { rps: '', timeout: '' })
  const saveLimits = () => {
    action(() => api.patchJob(id, {
      config: {
        global_rate_limit_rps: Number(limitsValue.rps),
        request_timeout_seconds: Number(limitsValue.timeout),
      },
    }))
  }

  // Assigned workers, threads are editable per worker
  const [assignment, setAssignment] = useState<Record<number, number> | null>(null)
  const assignmentValue = assignment ?? (job
    ? Object.fromEntries(workers.map((w) => [
        w.id,
        job.workers?.includes(w.id)
          ? (w.config.threads || 0)
          : (job.config.servers.find((s) => s.name === w.config.name)?.threads ?? w.config.threads ?? 0),
      ]))
    : {})
  const saveAssignment = () => {
    const selected = workers.filter((w) => assignmentValue[w.id] > 0)
    const threadPatches = selected
      .filter((w) => assignmentValue[w.id] !== w.config.threads)
      .map((w) => api.patchWorker(w.id, { config: { threads: assignmentValue[w.id] } }))
    const servers = job?.config.servers.map((s) => {
      const w = workers.find((x) => x.config.name === s.name)
      return w && assignmentValue[w.id] > 0 ? { ...s, threads: assignmentValue[w.id] } : s
    }) ?? []
    Promise.all([...threadPatches, api.patchJob(id, { workers: selected.map((w) => w.id), config: { servers } })])
      .then(() => { setAssignment(null); refresh() })
      .catch((e: any) => setErr(e.message))
  }

  // Filter and export of the downloaded results
  const filteredItems = items.filter((v) =>
    itemFilter === '' || String(v).includes(itemFilter))
  const exportCsv = () => {
    const rows = ['id', ...filteredItems.map(String)]
    const blob = new Blob([rows.join('\n')], { type: 'text/csv' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `job-${id}-videos.csv`
    a.click()
    URL.revokeObjectURL(a.href)
  }

  if (!job) return <div className="loading">{err || 'Loading...'}</div>

  const total = rangeSize(job.config.range.from, job.config.range.to)
  const processed = metrics?.requests ?? 0
  const pct = total == null ? null : total === 0 ? 100 : Math.min(100, (processed / total) * 100)
  const remaining = total == null ? null : Math.max(0, total - processed)
  const speed = metrics?.identifiers_per_second ?? 0

  return (
    <div>
      <div className="page-header">
        <h1>Job #{job.id} · {job.config.name}</h1>
        <Link to="/admin/jobs">← All jobs</Link>
      </div>

      {err && <div className="error">{err}</div>}

      <div className="stat-grid">
        <div className="stat"><div className="stat-value">{job.status}</div><div className="stat-label">Status</div></div>
        <div className="stat"><div className="stat-value">{total == null ? '∞' : total}</div><div className="stat-label">Range size</div></div>
        <div className="stat"><div className="stat-value">{processed}</div><div className="stat-label">Processed IDs</div></div>
        <div className="stat"><div className="stat-value">{metrics?.found_materials ?? '—'}</div><div className="stat-label">Materials found</div></div>
        <div className="stat"><div className="stat-value">{items.length}</div><div className="stat-label">Files downloaded</div></div>
        <div className="stat"><div className="stat-value">{metrics?.final_errors ?? '—'}</div><div className="stat-label">Errors</div></div>
        <div className="stat"><div className="stat-value">{fmtIds(speed)}</div><div className="stat-label">Check speed</div></div>
        <div className="stat"><div className="stat-value">{fmtBytesPerSec(metrics?.bytes_per_second)}</div><div className="stat-label">Download speed</div></div>
        <div className="stat"><div className="stat-value">{fmtEta(remaining, speed)}</div><div className="stat-label">ETA</div></div>
        <div className="stat"><div className="stat-value">{job.workers?.length ?? 0}</div><div className="stat-label">Assigned workers</div></div>
      </div>

      <div className="card">
        <h2>Progress</h2>
        <ProgressBar
          value={pct ?? 0}
          max={100}
          label={pct == null ? 'open range' : `${pct.toFixed(1)}% · ${processed}/${total == null ? '∞' : total}`}
        />
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
        <h2>Limits</h2>
        <div className="inline-controls">
          <label>Global rate limit (RPS)
            <input type="number" min={0} style={{ width: 120 }} value={limitsValue.rps}
              onChange={(e) => setLimits({ ...limitsValue, rps: e.target.value })} />
          </label>
          <label>Request timeout (s)
            <input type="number" min={1} style={{ width: 120 }} value={limitsValue.timeout}
              onChange={(e) => setLimits({ ...limitsValue, timeout: e.target.value })} />
          </label>
          <button onClick={saveLimits}>Save limits</button>
        </div>
      </div>

      <div className="card">
        <h2>Assigned servers</h2>
        <div className="table-wrap">
          <table className="table">
            <thead><tr><th>ID</th><th>Name</th><th>Host</th><th>Threads</th><th>Assigned</th></tr></thead>
            <tbody>
              {workers.map((w) => (
                <tr key={w.id}>
                  <td>#{w.id}</td>
                  <td>{w.config.name}</td>
                  <td>{w.config.host}</td>
                  <td>
                    <input type="number" min={0} style={{ width: 80 }}
                      value={assignmentValue[w.id] ?? 0}
                      onChange={(e) => setAssignment({ ...assignmentValue, [w.id]: Number(e.target.value) })} />
                  </td>
                  <td>
                    <input type="checkbox" checked={(assignmentValue[w.id] ?? 0) > 0}
                      onChange={(e) => setAssignment({
                        ...assignmentValue,
                        [w.id]: e.target.checked ? Math.max(1, assignmentValue[w.id] || w.config.threads || 1) : 0,
                      })} />
                  </td>
                </tr>
              ))}
              {workers.length === 0 && <tr><td colSpan={5} className="muted">No workers registered.</td></tr>}
            </tbody>
          </table>
        </div>
        <div className="form-actions">
          <button onClick={saveAssignment}>Save servers</button>
        </div>
      </div>

      <div className="card">
        <h2>Configuration</h2>
        <pre className="code">{JSON.stringify(job.config, null, 2)}</pre>
      </div>

      <div className="card">
        <div className="page-header">
          <h2>Downloaded videos ({items.length})</h2>
          <div className="row-actions">
            <input placeholder="Filter by id..." style={{ width: 140 }}
              value={itemFilter} onChange={(e) => setItemFilter(e.target.value)} />
            <button onClick={exportCsv} disabled={filteredItems.length === 0}>Export CSV</button>
          </div>
        </div>
        <div className="chip-grid">
          {filteredItems.slice(0, 200).map((vid) => (
            <Link key={vid} to={`/video/${vid}`} className="chip">#{vid}</Link>
          ))}
          {filteredItems.length > 200 && <span className="muted">…and {filteredItems.length - 200} more</span>}
          {filteredItems.length === 0 && <span className="muted">No videos downloaded yet.</span>}
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
