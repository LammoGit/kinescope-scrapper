import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../auth'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true); setErr(null)
    try {
      await login(email, password)
      navigate('/admin')
    } catch (e: any) {
      setErr(e.message || 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-card">
      <h1>Sign in</h1>
      <form onSubmit={submit}>
        <label>Email
          <input type="email" required value={email}
            onChange={(e) => setEmail(e.target.value)} />
        </label>
        <label>Password
          <input type="password" required minLength={8} value={password}
            onChange={(e) => setPassword(e.target.value)} />
        </label>
        {err && <div className="error">{err}</div>}
        <button type="submit" disabled={busy}>{busy ? 'Signing in...' : 'Sign in'}</button>
      </form>
      <p className="muted">
        No account? <Link to="/register">Register</Link>
      </p>
    </div>
  )
}
