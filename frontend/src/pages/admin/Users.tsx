import { useEffect, useState } from 'react'
import { api } from '../../api'
import type { PublicUser, UserRole } from '../../types'

export default function Users() {
  const [users, setUsers] = useState<PublicUser[]>([])
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState<Record<number, boolean>>({})
  // Fallback for environments where the list endpoint is unavailable:
  // individual users can still be managed by numeric ID.
  const [lookupId, setLookupId] = useState('')

  useEffect(() => {
    api.listUsers()
      .then(setUsers)
      .catch(() => { /* fall back to manual lookup below */ })
  }, [])

  const lookup = async () => {
    const id = Number(lookupId)
    if (!id) return
    try {
      const u = await api.getUser(id)
      setUsers((prev) => prev.find((x) => x.id === id) ? prev : [...prev, u])
    } catch (e: any) {
      setErr(e.message)
    }
  }

  const changeRole = async (id: number, role: UserRole) => {
    setBusy((b) => ({ ...b, [id]: true }))
    try {
      const updated = await api.changeRole(id, role)
      setUsers((prev) => prev.map((u) => u.id === id ? updated : u))
    } catch (e: any) { setErr(e.message) }
    finally { setBusy((b) => ({ ...b, [id]: false })) }
  }

  const remove = async (id: number) => {
    if (!confirm(`Delete user #${id}?`)) return
    try {
      await api.deleteUser(id)
      setUsers((prev) => prev.filter((u) => u.id !== id))
    } catch (e: any) { setErr(e.message) }
  }

  return (
    <div>
      <h1>Users</h1>
      {err && <div className="error">{err}</div>}

      <div className="card form-row">
        <label>Lookup by ID
          <input value={lookupId} onChange={(e) => setLookupId(e.target.value)} placeholder="42" />
        </label>
        <button onClick={lookup}>Lookup</button>
      </div>

      <table className="table">
        <thead><tr><th>ID</th><th>Username</th><th>Email</th><th>Role</th><th>Actions</th></tr></thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td>#{u.id}</td>
              <td>{u.username}</td>
              <td>{u.email}</td>
              <td>
                <select
                  value={u.role}
                  disabled={!!busy[u.id]}
                  onChange={(e) => changeRole(u.id, e.target.value as UserRole)}
                >
                  <option value="viewer">viewer</option>
                  <option value="operator">operator</option>
                  <option value="administrator">administrator</option>
                </select>
              </td>
              <td>
                <button onClick={() => remove(u.id)}>Delete</button>
              </td>
            </tr>
          ))}
          {users.length === 0 && <tr><td colSpan={5} className="muted">Look up a user by ID to manage it.</td></tr>}
        </tbody>
      </table>
    </div>
  )
}
