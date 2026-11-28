import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api'
import type { Video, VideoMetadata } from '../types'
import VideoCard from '../components/VideoCard'

export default function VideoPage() {
  const { videoId } = useParams()
  const id = Number(videoId)

  const [meta, setMeta] = useState<VideoMetadata | null>(null)
  const [url, setUrl] = useState<string>('')
  const [recommended, setRecommended] = useState<Video[]>([])
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!Number.isFinite(id) || id <= 0) {
      setErr('Invalid video id')
      return
    }
    setErr(null)
    api.getVideo(id)
      .then((r) => { setUrl(r.url); setMeta(r.metadata) })
      .catch((e) => setErr(e.message))

    api.getRecommendedForVideo(id, 12, 0)
      .then((r) => setRecommended(r.items || []))
      .catch(() => {})
  }, [id])

  return (
    <div className="video-page">
      <div className="video-main">
        <div className="player">
          {url ? (
            <video controls src={url} poster="">
              Your browser does not support the video tag.
            </video>
          ) : (
            <div className="player-placeholder">
              {err ? <span className="error">{err}</span> : 'Loading stream...'}
            </div>
          )}
        </div>

        {meta && (
          <div className="video-info">
            <h1>{meta.title}</h1>
            <p className="muted">
              {meta.mime_type} · {formatBytes(meta.file_size)} · {meta.status}
            </p>
            <p className="muted small">
              Source: <a href={meta.source_url} target="_blank" rel="noreferrer">{meta.source_url}</a>
            </p>
            <p className="muted small">
              Found at {new Date(meta.found_at).toLocaleString()} · Downloaded at {new Date(meta.downloaded_at).toLocaleString()}
            </p>
          </div>
        )}
      </div>

      <aside className="video-side">
        <h2>Recommended</h2>
        <div className="side-list">
          {recommended.length === 0 && <p className="muted">No recommendations.</p>}
          {recommended.map((v) => (
            <Link to={`/video/${v.id}`} key={v.id} className="side-item">
              <div className="side-thumb" />
              <div>
                <div className="side-title">{v.metadata.title || `Video #${v.id}`}</div>
                <div className="muted small">#{v.id}</div>
              </div>
            </Link>
          ))}
        </div>
      </aside>
    </div>
  )
}

function formatBytes(b: number): string {
  if (!b) return '0 B'
  const u = ['B', 'KiB', 'MiB', 'GiB']
  let i = 0, n = b
  while (n >= 1024 && i < u.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(1)} ${u[i]}`
}
