import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { AuthProvider } from './auth'
import { PublicLayout } from './components/Layout'
import AdminLayout from './components/AdminLayout'
import { RequireAuth, RequireRole } from './components/RequireAuth'

import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import VideoPage from './pages/VideoPage'
import Dashboard from './pages/admin/Dashboard'
import Jobs from './pages/admin/Jobs'
import JobDetail from './pages/admin/JobDetail'
import Workers from './pages/admin/Workers'
import Users from './pages/admin/Users'
import Events from './pages/admin/Events'
import Terminal from './pages/admin/Terminal'

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<PublicLayout />}>
            <Route path="/" element={<Home />} />
            <Route path="/video/:videoId" element={<VideoPage />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
          </Route>

          <Route path="/admin" element={<RequireAuth><AdminLayout /></RequireAuth>}>
            <Route index element={<Dashboard />} />
            <Route path="jobs" element={<Jobs />} />
            <Route path="jobs/:jobId" element={<JobDetail />} />
            <Route path="workers" element={<Workers />} />
            <Route path="events" element={<Events />} />
            <Route path="users" element={
              <RequireRole role="administrator"><Users /></RequireRole>
            } />
            <Route path="terminal/:workerId" element={
              <RequireRole role="administrator"><Terminal /></RequireRole>
            } />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
