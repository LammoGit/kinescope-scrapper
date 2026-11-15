import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../../api'
import type { Job, Worker, JobConfig } from '../../types'

const emptyServer = (): WorkerConfig => ({
  name: '', host: '', ssh_port: 22, ssh_user: '', ssh_key: '',
  threads: 4, storage_path: '/data/videos', min_free_space_gb: 10,
})

const emptyJob = (): JobConfig => ({
  name: '',
  range: { from: 0, to: 10000 },
  request_timeout_seconds: 20,
  max_tries: 4,
  global_rate_limit_rps: 20,
  stop_on_consecutive_missing: 0,
  servers: [emptyServer()],
})

export default function Jobs() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<Worker[]>([])
  const [err, setErr] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState<JobConfig>(emptyJob())
  const [busy, setBusy] = useState(false)
  const navigate = useNavigate()

  const refresh = () => {
    api.listJobs().then(setJobs).catch((e) => setErr(e.message))
    api.listWorkers().then(setWorkers).catch(() => {})
  }
  useEffect(refresh, [])

  const action = async (fn: () => Promise<any>) => {
    try { await fn(); refresh() }
    catch (e: any) { setErr(e.message) }
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true); setErr(null)
    try {
      const created = await api.createJob(form)
      setShowCreate(false)
      setForm(emptyJob())
      navigate(`/admin/jobs/${created.id}`)
    } catch (e: any) {
      setErr(e.message)
    } finally {
      setBusy(false)
    }
  }

  const loadWorkerConfigs = () => {
    setForm((f) => ({
      ...f,
      servers: workers.map((w) => ({ ...w.config })),
    }))
  }

  return (
    <div>
      <div className="page-header">
        <h1>Jobs</h1>
        <button onClick={() => setShowCreate((v) => !v)}>
          {showCreate ? 'Cancel' : 'New job'}
        </button>
      </div>

      {err && <div className="error">{err}</div>}

      {showCreate && (
        <form className="card form" onSubmit={submit}>
          <h2>Create job</h2>
          <div className="form-grid">
            <label>Name
              <input value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                required />
            </label>
            <label>From
              <input type="number" min={0} value={form.range.from}
                onChange={(e) => setForm({ ...form, range: { ...form.range, from: Number(e.target.value) } })} />
            </label>
            <label>To
              <input type="number" min={0} value={form.range.to}
                onChange={(e) => setForm({ ...form, range: { ...form.range, to: Number(e.target.value) } })} />
            </label>
            <label>Request timeout (s)
              <input type="number" min={1} value={form.request_timeout_seconds}
                onChange={(e) => setForm({ ...form, request_timeout_seconds: Number(e.target.value) })} />
            </label>
            <label>Max tries
              <input type="number" min={0} value={form.max_tries}
                onChange={(e) => setForm({ ...form, max_tries: Number(e.target.value) })} />
            </label>
            <label>Global rate limit (RPS)
              <input type="number" min={0} value={form.global_rate_limit_rps}
                onChange={(e) => setForm({ ...form, global_rate_limit_rps: Number(e.target.value) })} />
            </label>
            <label>Stop after consecutive missing
              <input type="number" min={0} value={form.stop_on_consecutive_missing}
                onChange={(e) => setForm({ ...form, stop_on_consecutive_missing: Number(e.target.value) })} />
            </label>
          </div>

          <h3>Servers</h3>
          {workers.length > 0 && (
            <div className="form-row">
              <button type="button" onClick={loadWorkerConfigs}>
                Load from {workers.length} registered worker(s)
              </button>
            </div>
          )}

          {form.servers.map((s, i) => (
            <div className="server-row" key={i}>
              <div className="form-grid">
                <label>Name
                  <input value={s.name} onChange={(e) => updateServer(i, { name: e.target.value })} required />
                </label>
                <label>Host
                  <input value={s.host} onChange={(e) => updateServer(i, { host: e.target.value })} required />
                </label>
                <label>SSH port
                  <input type="number" value={s.ssh_port} onChange={(e) => updateServer(i, { ssh_port: Number(e.target.value) })} />
                </label>
                <label>SSH user
                  <input value={s.ssh_user} onChange={(e) => updateServer(i, { ssh_user: e.target.value })} required />
                </label>
                <label>SSH key path
                  <input value={s.ssh_key} onChange={(e) => updateServer(i, { ssh_key: e.target.value })} required />
                </label>
                <label>Threads
                  <input type="number" value={s.threads} onChange={(e) => updateServer(i, { threads: Number(e.target.value) })} />
                </label>
                <label>Storage path
                  <input value={s.storage_path} onChange={(e) => updateServer(i, { storage_path: e.target.value })} />
                </label>
                <label>Min free space (GB)
                  <input type="number" value={s.min_free_space_gb} onChange={(e) => updateServer(i, { min_free_space_gb: Number(e.target.value) })} />
                </label>
              </div>
              <button type="button" onClick={() => removeServer(i)}>Remove</button>
            </div>
          ))}
          <button type="button" onClick={() => setForm({ ...form, servers: [...form.servers, emptyServer()] })}>
            Add server
          </button>

          <div className="form-actions">
            <button type="submit" disabled={busy}>{busy ? 'Creating...' : 'Create job'}</button>
          </div>
        </form>
      )}

      <table className="table">
        <thead><tr><th>ID</th><th>Name</th><th>Status</th><th>Range</th><th>Actions</th></tr></thead>
        <tbody>
          {jobs.map((j) => (
            <tr key={j.id}>
              <td><Link to={`/admin/jobs/${j.id}`}>#{j.id}</Link></td>
              <td>{j.config.name}</td>
              <td><span className={`pill status-${j.status}`}>{j.status}</span></td>
              <td>{j.config.range.from} – {j.config.range.to}</td>
              <td className="row-actions">
                <button onClick={() => action(() => api.startJob(j.id))}>Start</button>
                <button onClick={() => action(() => api.pauseJob(j.id))}>Pause</button>
                <button onClick={() => action(() => api.resumeJob(j.id))}>Resume</button>
                <button onClick={() => action(() => api.stopJob(j.id))}>Stop</button>
                <button onClick={() => { if (confirm('Delete job?')) action(() => api.deleteJob(j.id)) }}>Delete</button>
              </td>
            </tr>
          ))}
          {jobs.length === 0 && <tr><td colSpan={5} className="muted">No jobs yet</td></tr>}
        </tbody>
      </table>
    </div>
  )

  function updateServer(i: number, patch: Partial<WorkerConfig>) {
    setForm((f) => ({ ...f, servers: f.servers.map((s, j) => j === i ? { ...s, ...patch } : s) }))
  }
  function removeServer(i: number) {
    setForm((f) => ({ ...f, servers: f.servers.filter((_, j) => j !== i) }))
  }
}
