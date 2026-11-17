import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, EventStream } from '../../api'
import type { Worker, Metrics } from '../../types'
import { fmtBytes, fmtBytesPerSec, fmtIds, fmtUptime } from '../../format'

export default function Workers() {
  const [workers, setWorkers] = useState<Worker[]>([])
  const [metrics, setMetrics] = useState<Record<number, Metrics>>({})
  const [err, setErr] = useState<string | null>(null)
  const [showRegister, setShowRegister] = useState(false)
  const [form, setForm] = useState({
    name: '', host: '', ssh_port: 22, ssh_user: '', ssh_password: '',
    threads: 8, storage_path: '/data/videos', min_free_space_gb: 20,
  })
  const [busy, setBusy] = useState(false)

  const refresh = () => {
    api.listWorkers().then(setWorkers).catch((e) => setErr(e.message))
  }
  useEffect(refresh, [])

  // Live metrics
  useEffect(() => {
    const es = new EventStream('/stream/workers/metrics')
    es.addEventListener('worker.metrics', (e) => {
      try {
        const list = api.asArray<Metrics>(JSON.parse(e.data))
        setMetrics((prev) => {
          const next = { ...prev }
          for (const m of list) next[m.worker_id] = m
          return next
        })
      } catch {}
    })
    return () => es.close()
  }, [])

  // Live worker status changes (registered, deleted, state changes)
  useEffect(() => {
    const es = new EventStream('/stream/workers/updates')
    const onUpdate = (e: MessageEvent) => {
      try {
        const list = api.asArray<Worker>(JSON.parse(e.data))
        setWorkers((prev) => {
          let next = [...prev]
          for (const w of list) {
            if (w?.id == null) continue
            const idx = next.findIndex((x) => x.id === w.id)
            // Only merge when we actually got a full worker object
            if (w.config) {
              if (idx >= 0) next[idx] = w
              else next = [...next, w]
            }
          }
          return next
        })
      } catch {}
    }
    const onDelete = (e: MessageEvent) => {
      try {
        const { id } = JSON.parse(e.data)
        setWorkers((prev) => prev.filter((w) => w.id !== id))
      } catch {}
    }
    es.addEventListener('worker.update', onUpdate)
    es.addEventListener('worker.delete', onDelete)
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

      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th><th>Name</th><th>Host</th><th>Status</th><th>Block</th>
              <th>Threads</th><th>Checks</th><th>Downloads</th>
              <th>CPU</th><th>RAM</th><th>Disk load</th><th>Free space</th>
              <th>Network</th><th>Download speed</th><th>IDs/s</th>
              <th>Errors</th><th>Retries</th><th>Uptime</th><th>Actions</th>
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
                  <td>{m && m.range_id > 0 ? `#${m.range_id}` : '—'}</td>
                  <td>{w.config.threads}</td>
                  <td>{m?.active_connections ?? '—'}</td>
                  <td>{m?.active_downloads ?? '—'}</td>
                  <td>{m ? `${(m.cpu_load * 100).toFixed(0)}%` : '—'}</td>
                  <td>{m ? `${(m.memory_load * 100).toFixed(0)}%` : '—'}</td>
                  <td>{m ? `${(m.disk_load * 100).toFixed(0)}%` : '—'}</td>
                  <td>{m ? fmtBytes(m.free_space_bytes) : '—'}</td>
                  <td>{m ? `${(m.network_load * 100).toFixed(0)}%` : '—'}</td>
                  <td>{fmtBytesPerSec(m?.bytes_per_second)}</td>
                  <td>{fmtIds(m?.identifiers_per_second)}</td>
                  <td>{m?.final_errors ?? '—'}</td>
                  <td>{m?.repeated_attempts ?? '—'}</td>
                  <td>{fmtUptime(m?.worker_uptime)}</td>
                  <td className="row-actions">
                    <Link to={`/admin/terminal/${w.id}`}><button>SSH</button></Link>
                    <button onClick={() => { if (confirm(`Delete worker #${w.id}?`)) api.deleteWorker(w.id).then(refresh).catch((e) => setErr(e.message)) }}>
                      Delete
                    </button>
                  </td>
                </tr>
              )
            })}
            {workers.length === 0 && <tr><td colSpan={19} className="muted">No workers registered.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  )
}
