import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../../api'
import type { Job, Worker, Metrics } from '../../types'

interface LiveJob extends Job { metrics?: Metrics }

export default function Dashboard() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<Worker[]>([])
  const [metrics, setMetrics] = useState<Record<number, Metrics>>({})
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    api.listJobs().then(setJobs).catch((e) => setErr(e.message))
    api.listWorkers().then(setWorkers).catch(() => {})

    const es = new EventSource('/api/stream/workers/metrics')
    es.addEventListener('worker.metrics', (e: MessageEvent) => {
      try {
        const list: Metrics[] = JSON.parse(e.data)
        const next: Record<number, Metrics> = {}
        for (const m of list) next[m.worker_id] = m
        setMetrics(next)
      } catch {}
    })
    return () => es.close()
  }, [])

  const running = jobs.filter((j) => j.status === 'running').length
  const activeWorkers = workers.filter((w) => w.status === 'working').length
  const totalFound = Object.values(metrics).reduce((a, m) => a + m.found_materials, 0)
  const totalDl = Object.values(metrics).reduce((a, m) => a + m.active_downloads, 0)
  const avgCpu = Object.values(metrics).length
    ? Object.values(metrics).reduce((a, m) => a + m.cpu_load, 0) / Object.values(metrics).length
    : 0

  return (
    <div>
      <h1>Dashboard</h1>
      {err && <div className="error">{err}</div>}

      <div className="stat-grid">
        <Stat label="Jobs" value={jobs.length} />
        <Stat label="Running" value={running} />
        <Stat label="Workers" value={workers.length} />
        <Stat label="Active workers" value={activeWorkers} />
        <Stat label="Materials found" value={totalFound} />
        <Stat label="Active downloads" value={totalDl} />
        <Stat label="Avg CPU" value={`${(avgCpu * 100).toFixed(1)}%`} />
      </div>

      <h2>Recent jobs</h2>
      <table className="table">
        <thead><tr><th>ID</th><th>Name</th><th>Status</th><th>Range</th><th>Workers</th></tr></thead>
        <tbody>
          {jobs.slice(0, 10).map((j) => (
            <tr key={j.id}>
              <td><Link to={`/admin/jobs/${j.id}`}>#{j.id}</Link></td>
              <td>{j.config.name}</td>
              <td><span className={`pill status-${j.status}`}>{j.status}</span></td>
              <td>{j.config.range.from} – {j.config.range.to}</td>
              <td>{j.workers?.length ?? 0}</td>
            </tr>
          ))}
          {jobs.length === 0 && <tr><td colSpan={5} className="muted">No jobs</td></tr>}
        </tbody>
      </table>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: any }) {
  return (
    <div className="stat">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}
