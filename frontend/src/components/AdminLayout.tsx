import { NavLink, Outlet, useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../auth'

export default function AdminLayout() {
  const { user, logout, isAdmin } = useAuth()
  const navigate = useNavigate()

  return (
    <div className="admin">
      <aside className="sidebar">
        <div className="brand"><Link to="/">Scrappy</Link></div>
        <nav>
          <NavLink to="/admin" end>Dashboard</NavLink>
          <NavLink to="/admin/jobs">Jobs</NavLink>
          <NavLink to="/admin/workers">Workers</NavLink>
          <NavLink to="/admin/events">Events</NavLink>
          {isAdmin && <NavLink to="/admin/users">Users</NavLink>}
        </nav>
        <div className="sidebar-user">
          <div><strong>{user?.role}</strong></div>
          <div>id: {user?.id}</div>
          <button onClick={() => { logout(); navigate('/') }}>Logout</button>
        </div>
      </aside>
      <main className="admin-main">
        <Outlet />
      </main>
    </div>
  )
}
