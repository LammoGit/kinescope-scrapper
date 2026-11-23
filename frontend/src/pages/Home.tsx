import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api'
import type { Video } from '../types'
import VideoCard from '../components/VideoCard'

export default function Home() {
  const [params] = useSearchParams()
  const query = params.get('q') || ''

  const [videos, setVideos] = useState<Video[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    const fetchData = query
      ? api.search(query, 24, 0)
      : api.recommended(24, 0)

    fetchData
      .then((r) => { if (!cancelled) setVideos(r.items || []) })
      .catch((e) => { if (!cancelled) { setError(e.message); setVideos([]) } })
      .finally(() => { if (!cancelled) setLoading(false) })

    return () => { cancelled = true }
  }, [query])

  return (
    <div className="home">
      <div className="hero">
        <h1>{query ? `Results for "${query}"` : 'Recommended Videos'}</h1>
        <p className="muted">
          Distributed archive of video materials discovered across a numerical ID range.
        </p>
      </div>

      {loading && <div className="loading">Loading videos...</div>}
      {error && <div className="error">{error}</div>}

      {!loading && !error && videos.length === 0 && (
        <div className="empty">No videos found.</div>
      )}

      <div className="video-grid">
        {videos.map((v) => <VideoCard key={v.id} video={v} />)}
      </div>
    </div>
  )
}
