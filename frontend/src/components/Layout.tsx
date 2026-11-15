import { Link, Outlet, useNavigate, useSearchParams } from 'react-router-dom'
import { useState } from 'react'
import { useAuth } from '../auth'

export function PublicLayout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [q, setQ] = useState(params.get('q') || '')

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const query = q.trim()
    if (query) navigate(`/?q=${encodeURIComponent(query)}`)
    else navigate('/')
  }

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">Kinescope Archive</Link>
        <form className="searchbar" onSubmit={submit}>
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search videos..."
          />
          <button type="submit">Search</button>
        </form>
        <nav className="topnav">
          {user ? (
            <>
              <Link to="/admin">Admin</Link>
              <button onClick={() => { logout(); navigate('/') }}>Logout</button>
            </>
          ) : (
            <>
              <Link to="/login">Login</Link>
              <Link to="/register">Register</Link>
            </>
          )}
        </nav>
      </header>
      <main className="container">
        <Outlet />
      </main>
    </div>
  )
}
