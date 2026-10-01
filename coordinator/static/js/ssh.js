const workerId = decodeURIComponent(window.location.pathname.split('/').filter(Boolean)[1] || '');
document.getElementById('ssh-worker-id').textContent = workerId || '—';

const statusEl = document.getElementById('ssh-status');

const term = new Terminal({
  cursorBlink: true,
  fontFamily: 'ui-monospace, SFMono-Regular, "JetBrains Mono", Consolas, monospace',
  fontSize: 13,
  theme: {
    background: '#000000',
    foreground: '#e6edf3',
    cursor: '#58a6ff',
  },
});
const fit = new FitAddon.FitAddon();
term.loadAddon(fit);
term.open(document.getElementById('terminal'));
fit.fit();

window.addEventListener('resize', () => { try { fit.fit(); } catch (_) {} });

function setStatus(text, cls) {
  statusEl.textContent = text;
  statusEl.className = 'badge ' + (cls || 'badge-idle');
}

const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
const wsUrl = `${proto}://${window.location.host}/ssh/${encodeURIComponent(workerId)}/ws`;
let ws = null;
let idleTimer = null;
const IDLE_MS = 15 * 60 * 1000;

function resetIdle() {
  clearTimeout(idleTimer);
  idleTimer = setTimeout(() => {
    term.writeln('\r\n\x1b[33m[disconnected: idle timeout]\x1b[0m');
    try { ws && ws.close(); } catch (_) {}
  }, IDLE_MS);
}

function connect() {
  setStatus('connecting…', 'badge-idle');
  ws = new WebSocket(wsUrl);
  ws.binaryType = 'arraybuffer';

  ws.onopen = () => {
    setStatus('connected', 'badge-running');
    term.focus();
    resetIdle();
    sendResize();
  };

  ws.onmessage = (ev) => {
    if (typeof ev.data === 'string') term.write(ev.data);
    else term.write(new Uint8Array(ev.data));
    resetIdle();
  };

  ws.onclose = (ev) => {
    setStatus('closed', 'badge-failed');
    term.writeln(`\r\n\x1b[31m[connection closed${ev.reason ? ': ' + ev.reason : ''}]\x1b[0m`);
  };

  ws.onerror = () => {
    setStatus('error', 'badge-failed');
    toast('SSH connection error', 'error');
  };
}

function sendResize() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
  }
}

term.onData((data) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'input', data }));
    resetIdle();
  }
});

term.onResize(sendResize);

document.getElementById('terminal').addEventListener('click', () => term.focus());

connect();
