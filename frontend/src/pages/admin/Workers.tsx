import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../../api'
import type { Worker, Metrics } from '../../types'

export default function Workers() {
  const [workers, setWorkers] = useState<Worker[]>([])
  const [metrics, setMetrics] = useState<Record<number, Metrics>>({})
  const [err, setErr] = useState<string | null>(null)
  const [showRegister, setShowRegister] = useState(false)
  const [form, setForm] = useState({
    name: '', host: '', ssh_port: 22, ssh_user: '', ssh_key: '', ssh_password: '',
    threads: 8, storage_path: '/data/videos', min_free_space_gb: 20,
  })
  const [busy, setBusy] = useState(false)

  const refresh = () => {
    api.listWorkers().then(setWorkers).catch((e) => setErr(e.message))
    for (const w of workers) {
      api.getWorkerMetrics(w.id).then((m) => setMetrics((prev) => ({ ...prev, [w.id]: m }))).catch(() => {})
    }
  }
  useEffect(refresh, [])
  useEffect(() => { refresh() /* re-fetch metrics when list updates */ }, [workers.length])

  // Live updates
  useEffect(() => {
    const es = new EventSource('/api/stream/workers/metrics')
    es.addEventListener('worker.metrics', (e: MessageEvent) => {
      try {
        const list: Metrics[] = JSON.parse(e.data)
        const next = { ...metrics }
        for (const m of list) next[m.worker_id] = m
        setMetrics(next)
      } catch {}
    })
    return () => es.close()
  }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true); setErr(null)
    try {
      await api.registerWorker(form)
      setShowRegister(false)
      refresh()
    } catch (e: any) {
      setErr(e.message)
    } finally { setBusy(false) }
  }

  return (
    <div>
      <div className="page-header">
        <h1>Workers</h1>
        <button onClick={() => setShowRegister((v) => !v)}>
          {showRegister ? 'Cancel' : 'Register worker'}
        </button>
      </div>

      {err && <div className="error">{err}</div>}

      {showRegister && (
        <form className="card form" onSubmit={submit}>
          <h2>Register a new worker</h2>
          <div className="form-grid">
            <label>Name
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </label>
            <label>Host
              <input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} required />
            </label>
            <label>SSH port
              <input type="number" value={form.ssh_port} onChange={(e) => setForm({ ...form, ssh_port: Number(e.target.value) })} required />
            </label>
            <label>SSH user
              <input value={form.ssh_user} onChange={(e) => setForm({ ...form, ssh_user: e.target.value })} required />
            </label>
            <label>SSH key path
              <input value={form.ssh_key} onChange={(e) => setForm({ ...form, ssh_key: e.target.value })} required />
            </label>
            <label>SSH password (optional)
              <input type="password" value={form.ssh_password} onChange={(e) => setForm({ ...form, ssh_password: e.target.value })} />
            </label>
            <label>Threads
              <input type="number" value={form.threads} onChange={(e) => setForm({ ...form, threads: Number(e.target.value) })} required />
            </label>
            <label>Storage path
              <input value={form.storage_path} onChange={(e) => setForm({ ...form, storage_path: e.target.value })} required />
            </label>
            <label>Min free space (GB)
              <input type="number" value={form.min_free_space_gb} onChange={(e) => setForm({ ...form, min_free_space_gb: Number(e.target.value) })} required />
            </label>
          </div>
          <div className="form-actions">
            <button type="submit" disabled={busy}>{busy ? 'Provisioning...' : 'Register'}</button>
          </div>
        </form>
      )}

      <table className="table">
        <thead>
          <tr>
            <th>ID</th><th>Name</th><th>Host</th><th>Status</th>
            <th>CPU</th><th>MEM</th><th>Disk</th><th>IDs/s</th><th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {workers.map((w) => {
            const m = metrics[w.id]
            return (
              <tr key={w.id}>
                <td>#{w.id}</td>
                <td>{w.config.name}</td>
                <td>{w.config.host}:{w.config.ssh_port}</td>
                <td><span className={`pill status-${w.status}`}>{w.status}</span></td>
                <td>{m ? `${(m.cpu_load * 100).toFixed(1)}%` : '—'}</td>
                <td>{m ? `${(m.memory_load * 100).toFixed(1)}%` : '—'}</td>
                <td>{m ? `${(m.disk_load * 100).toFixed(1)}%` : '—'}</td>
                <td>{m ? m.identifiers_per_second.toFixed(1) : '—'}</td>
                <td className="row-actions">
                  <Link to={`/admin/terminal/${w.id}`}><button>SSH</button></Link>
                  <button onClick={() => { if (confirm(`Delete worker #${w.id}?`)) api.deleteWorker(w.id).then(refresh).catch((e) => setErr(e.message)) }}>
                    Delete
                  </button>
                </td>
              </tr>
            )
          })}
          {workers.length === 0 && <tr><td colSpan={9} className="muted">No workers registered.</td></tr>}
        </tbody>
      </table>
    </div>
  )
}
