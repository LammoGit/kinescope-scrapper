import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, EventStream } from '../../api'
import type { Job, Worker, Metrics } from '../../types'
import ProgressBar from '../../components/ProgressBar'
import { fmtIds, fmtBytesPerSec, fmtEta, fmtRange, rangeSize } from '../../format'

// Sum every metric row belonging to one entity into a single aggregate
function aggregate(rows: Metrics[]): Metrics | null {
  if (rows.length === 0) return null
  const n = rows.length
  const sum = (pick: (m: Metrics) => number) => rows.reduce((a, m) => a + (pick(m) || 0), 0)
  return {
    range_id: 0,
    job_id: rows[0].job_id,
    worker_id: rows[0].worker_id,
    cpu_load: sum((m) => m.cpu_load) / n,
    memory_load: sum((m) => m.memory_load) / n,
    disk_load: sum((m) => m.disk_load) / n,
    network_load: sum((m) => m.network_load) / n,
    free_space_bytes: Math.min(...rows.map((m) => m.free_space_bytes ?? Infinity)),
    opened_files: sum((m) => m.opened_files),
    active_connections: sum((m) => m.active_connections),
    worker_uptime: Math.max(...rows.map((m) => m.worker_uptime ?? 0)),
    identifiers_per_second: sum((m) => m.identifiers_per_second),
    requests: sum((m) => m.requests),
    found_materials: sum((m) => m.found_materials),
    active_downloads: sum((m) => m.active_downloads),
    bytes_per_second: sum((m) => m.bytes_per_second),
    avg_check_speed: sum((m) => m.avg_check_speed) / n,
    avg_download_speed: sum((m) => m.avg_download_speed) / n,
    download_queue_size: sum((m) => m.download_queue_size),
    repeated_attempts: sum((m) => m.repeated_attempts),
    final_errors: sum((m) => m.final_errors),
    oldest_in_queue: Math.max(...rows.map((m) => m.oldest_in_queue ?? 0)),
  }
}

export default function Dashboard() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<Worker[]>([])
  const [downloaded, setDownloaded] = useState<Record<number, number>>({})
  const [jobMetrics, setJobMetrics] = useState<Record<number, Metrics[]>>({})
  const [workerMetrics, setWorkerMetrics] = useState<Record<number, Metrics>>({})
  const [err, setErr] = useState<string | null>(null)

  const refreshJobs = () => {
    api.listJobs().then((js) => {
      setJobs(js)
      // Downloaded file count per job, loaded once per job list refresh
      Promise.all(
        js.map((j) =>
          api
            .getJobItems(j.id)
            .then((items) => [j.id, items.length] as const)
            .catch(() => [j.id, 0] as const),
        ),
      ).then((entries) => setDownloaded(Object.fromEntries(entries)))
    }).catch((e) => setErr(e.message))
  }

  const refreshWorkers = () => {
    api.listWorkers().then(setWorkers).catch(() => {})
  }

  useEffect(() => {
    refreshJobs()
    refreshWorkers()

    // Per-job metrics rows, keyed by job
    const jobsEs = new EventStream('/stream/jobs/metrics')
    jobsEs.addEventListener('job.metrics', (e) => {
      try {
        const rows = api.asArray<Metrics>(JSON.parse(e.data))
        setJobMetrics((prev) => {
          const next = { ...prev }
          for (const row of rows) {
            if (row?.job_id == null) continue
            const list = next[row.job_id] ? [...next[row.job_id]] : []
            // A live row replaces the snapshot row of the same worker
            const idx = list.findIndex((m) => m.worker_id === row.worker_id)
            if (idx >= 0) list[idx] = row
            else list.push(row)
            next[row.job_id] = list
          }
          return next
        })
      } catch {}
    })

    // Per-worker metrics for the global totals
    const workersEs = new EventStream('/stream/workers/metrics')
    workersEs.addEventListener('worker.metrics', (e) => {
      try {
        const rows = api.asArray<Metrics>(JSON.parse(e.data))
        setWorkerMetrics((prev) => {
          const next = { ...prev }
          for (const m of rows) next[m.worker_id] = m
          return next
        })
      } catch {}
    })

    // Job table updates
    const updatesEs = new EventStream('/stream/jobs/updates')
    updatesEs.addEventListener('job.update', refreshJobs)
    updatesEs.addEventListener('job.state_change', refreshJobs)
    updatesEs.addEventListener('job.create', refreshJobs)
    updatesEs.addEventListener('job.delete', refreshJobs)

    // Worker status changes
    const workerUpdatesEs = new EventStream('/stream/workers/updates')
    workerUpdatesEs.addEventListener('worker.update', refreshWorkers)
    workerUpdatesEs.addEventListener('worker.delete', refreshWorkers)

    // Keep the tables fresh even when no stream event arrives
    const poll = setInterval(() => {
      refreshJobs()
      refreshWorkers()
    }, 5000)

    return () => {
      jobsEs.close()
      workersEs.close()
      updatesEs.close()
      workerUpdatesEs.close()
      clearInterval(poll)
    }
  }, [])

  const allWorkerMetrics = Object.values(workerMetrics)
  const global = aggregate(allWorkerMetrics)
  const activeServers = workers.filter((w) => w.status === 'working').length
  const activeThreads = workers
    .filter((w) => w.status === 'working')
    .reduce((a, w) => a + (w.config.threads || 0), 0)

  const jobAgg = (id: number): Metrics | null => aggregate(jobMetrics[id] || [])
  const processed = (id: number): number => jobAgg(id)?.requests ?? 0
  const speed = (id: number): number => jobAgg(id)?.identifiers_per_second ?? 0
  const progressPct = (j: Job): number | null => {
    const total = rangeSize(j.config.range.from, j.config.range.to)
    if (total == null) return null
    if (total === 0) return 100
    return Math.min(100, (processed(j.id) / total) * 100)
  }
  const eta = (j: Job): string => {
    const total = rangeSize(j.config.range.from, j.config.range.to)
    const remaining = total == null ? null : Math.max(0, total - processed(j.id))
    return fmtEta(remaining, speed(j.id))
  }

  return (
    <div>
      <h1>Dashboard</h1>
      {err && <div className="error">{err}</div>}

      <div className="stat-grid">
        <Stat label="Jobs" value={jobs.length} />
        <Stat label="Running" value={jobs.filter((j) => j.status === 'running').length} />
        <Stat label="Active servers" value={activeServers} />
        <Stat label="Active threads" value={activeThreads} />
        <Stat label="Processed IDs" value={global?.requests ?? '—'} />
        <Stat label="Materials found" value={global?.found_materials ?? '—'} />
        <Stat label="Files downloaded" value={
          Object.values(downloaded).reduce((a, n) => a + n, 0)
        } />
        <Stat label="Errors" value={global?.final_errors ?? '—'} />
        <Stat label="Check speed" value={fmtIds(global?.identifiers_per_second)} />
        <Stat label="Avg check speed" value={fmtIds(global?.avg_check_speed)} />
        <Stat label="Download speed" value={fmtBytesPerSec(global?.bytes_per_second)} />
        <Stat label="Avg download speed" value={fmtBytesPerSec(global?.avg_download_speed)} />
      </div>

      <h2>Jobs</h2>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th><th>Name</th><th>Status</th><th>Range</th>
              <th style={{ minWidth: 180 }}>Progress</th>
              <th>Processed</th><th>Found</th><th>Files</th><th>Errors</th>
              <th>Check speed</th><th>Download</th><th>ETA</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((j) => {
              const agg = jobAgg(j.id)
              const pct = progressPct(j)
              return (
                <tr key={j.id}>
                  <td><Link to={`/admin/jobs/${j.id}`}>#{j.id}</Link></td>
                  <td>{j.config.name}</td>
                  <td><span className={`pill status-${j.status}`}>{j.status}</span></td>
                  <td>{fmtRange(j.config.range.from, j.config.range.to)}</td>
                  <td>
                    <ProgressBar
                      value={pct ?? 0}
                      max={100}
                      label={pct == null ? '∞' : `${pct.toFixed(1)}%`}
                    />
                  </td>
                  <td>{processed(j.id)}</td>
                  <td>{agg?.found_materials ?? '—'}</td>
                  <td>{downloaded[j.id] ?? '—'}</td>
                  <td>{agg?.final_errors ?? '—'}</td>
                  <td>{fmtIds(agg?.identifiers_per_second)}</td>
                  <td>{fmtBytesPerSec(agg?.bytes_per_second)}</td>
                  <td>{eta(j)}</td>
                </tr>
              )
            })}
            {jobs.length === 0 && <tr><td colSpan={12} className="muted">No jobs</td></tr>}
          </tbody>
        </table>
      </div>
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
