import { Link } from 'react-router-dom'
import type { Video } from '../types'

export default function VideoCard({ video }: { video: Video }) {
  const m = video.metadata
  return (
    <Link to={`/video/${video.id}`} className="video-card">
      <div className="video-thumb">
        <span className="video-badge">{m.status}</span>
      </div>
      <div className="video-body">
        <h3>{m.title || `Video #${video.id}`}</h3>
        <p className="muted">{m.mime_type} · {formatBytes(m.file_size)}</p>
        <p className="muted small">#{video.id}</p>
      </div>
    </Link>
  )
}

function formatBytes(b: number): string {
  if (!b) return '0 B'
  const u = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let i = 0, n = b
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(1)} ${u[i]}`
}
