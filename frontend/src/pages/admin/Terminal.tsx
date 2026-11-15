import { useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { Terminal as XTerm } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'

export default function Terminal() {
  const { workerId } = useParams()
  const termRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!termRef.current) return
    const term = new XTerm({
      fontFamily: 'Menlo, Consolas, monospace',
      fontSize: 14,
      theme: { background: '#0b0d12' },
      cursorBlink: true,
    })
    term.open(termRef.current)

    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    // NOTE: The current backend `bearerAuthMiddleware` reads the Authorization
    // header, which browsers cannot attach to a WebSocket handshake. A common
    // pattern is to accept a short-lived signed token in the query string.
    // We send it here so a corresponding backend change can enable this view.
    const token = localStorage.getItem('access_token') || ''
    const ws = new WebSocket(
      `${proto}//${location.host}/api/ssh/${workerId}?token=${encodeURIComponent(token)}`,
    )
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      term.writeln(`Connected to worker #${workerId}\r\n`)
    }
    ws.onmessage = (e) => {
      if (typeof e.data === 'string') term.write(e.data)
      else term.write(new Uint8Array(e.data))
    }
    ws.onclose = () => term.writeln('\r\n[disconnected]')
    ws.onerror = () => term.writeln('\r\n[connection error]')

    const dataDisp = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(data)
    })

    const resizeDisp = term.onResize(() => { /* future: send resize frames */ })

    const resize = () => term.fit?.()
    window.addEventListener('resize', resize)

    return () => {
      window.removeEventListener('resize', resize)
      dataDisp.dispose()
      resizeDisp.dispose()
      ws.close()
      term.dispose()
    }
  }, [workerId])

  return (
    <div>
      <div className="page-header">
        <h1>SSH · Worker #{workerId}</h1>
        <Link to="/admin/workers">← Workers</Link>
      </div>
      <div className="terminal-wrap" ref={termRef} />
    </div>
  )
}
