import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../auth'

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true); setErr(null)
    try {
      await register(username, email, password)
      navigate('/login')
    } catch (e: any) {
      setErr(e.message || 'Registration failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-card">
      <h1>Create account</h1>
      <form onSubmit={submit}>
        <label>Username
          <input required minLength={3} value={username}
            onChange={(e) => setUsername(e.target.value)} />
        </label>
        <label>Email
          <input type="email" required value={email}
            onChange={(e) => setEmail(e.target.value)} />
        </label>
        <label>Password
          <input type="password" required minLength={8} value={password}
            onChange={(e) => setPassword(e.target.value)} />
        </label>
        {err && <div className="error">{err}</div>}
        <button type="submit" disabled={busy}>{busy ? 'Creating...' : 'Create account'}</button>
      </form>
      <p className="muted">
        Already have an account? <Link to="/login">Sign in</Link>
      </p>
    </div>
  )
}
