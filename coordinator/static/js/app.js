/* ============================ utilities ============================ */

function escapeHtml(v) {
  if (v == null) return '';
  return String(v)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function fmtNum(v) {
  if (v == null || v === '') return '0';
  const n = Number(v);
  if (!isFinite(n)) return '0';
  return n.toLocaleString();
}

function fmtBytes(v) {
  if (v == null || v === 0) return '0 B';
  const n = Number(v);
  if (!isFinite(n)) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let i = 0;
  let x = Math.abs(n);
  while (x >= 1024 && i < units.length - 1) { x /= 1024; i++; }
  const sign = n < 0 ? '-' : '';
  return `${sign}${x.toFixed(x < 10 && i > 0 ? 1 : 0)} ${units[i]}`;
}

// Bytes per second → human string.
function fmtSpeed(bps) {
  if (!bps) return '0 B/s';
  return fmtBytes(bps) + '/s';
}

function fmtDuration(sec) {
  if (sec == null || !isFinite(sec) || sec < 0) return '—';
  sec = Math.floor(sec);
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (d) return `${d}d ${h}h`;
  if (h) return `${h}h ${m}m`;
  if (m) return `${m}m ${s}s`;
  return `${s}s`;
}

function fmtTime(t) {
  if (!t) return '—';
  const d = (typeof t === 'number' || /^\d+$/.test(String(t))) ? new Date(Number(t)) : new Date(t);
  if (isNaN(d.getTime())) return String(t);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function timeAgo(t) {
  if (!t) return 'never';
  const then = (typeof t === 'number' || /^\d+$/.test(String(t)))
    ? Number(t) * (String(t).length <= 10 ? 1000 : 1)   // seconds vs ms heuristic
    : new Date(t).getTime();
  const diff = Math.max(0, Date.now() - then) / 1000;
  if (diff < 5)     return 'just now';
  if (diff < 60)    return `${Math.floor(diff)}s ago`;
  if (diff < 3600)  return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function shortId(id) {
  if (id == null) return '—';
  const s = String(id);
  return s.length > 8 ? s.slice(0, 8) : s;
}

function statusBadge(status) {
  const s = String(status || '').toLowerCase();
  const map = {
    running:   'badge-running',
    active:    'badge-running',
    working:   'badge-running',
    paused:    'badge-paused',
    idle:      'badge-idle',
    created:   'badge-idle',
    pending:   'badge-idle',
    completed: 'badge-completed',
    finished:  'badge-completed',
    done:      'badge-completed',
    stopped:   'badge-failed',
    failed:    'badge-failed',
    error:     'badge-failed',
  };
  const cls = map[s] || 'badge-idle';
  return `<span class="badge ${cls}">${escapeHtml(s || 'unknown')}</span>`;
}

/* ============================ toast ============================ */

function toast(message, kind = 'info', timeoutMs = 4000) {
  const box = document.getElementById('toast-container');
  if (!box) return;
  const el = document.createElement('div');
  el.className = `toast toast-${kind}`;
  el.textContent = message;
  box.appendChild(el);
  setTimeout(() => {
    el.classList.add('toast-leaving');
    setTimeout(() => el.remove(), 250);
  }, timeoutMs);
}

/* ============================ api helper ============================ */
/* All backend routes are mounted under /api. */
const API_BASE = '/api';

async function api(path, opts = {}) {
  const url = path.startsWith('http') ? path : API_BASE + path;
  const res = await fetch(url, {
    credentials: 'same-origin',
    ...opts,
    headers: {
      Accept: 'application/json',
      ...(opts.headers || {}),
    },
  });

  const ct = res.headers.get('content-type') || '';
  const body = ct.includes('application/json') ? await res.json().catch(() => null) : await res.text();

  if (!res.ok) {
    const msg = (body && body.error) || (typeof body === 'string' && body) || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return body;
}

/* ============================ SSE helper ============================ */
const DEFAULT_EVENTS = [
  'snapshot',
  'progress',
  'event',
  'error',
  'worker_registered',
  'worker_metrics',
  'worker_status',
  'worker_updated',
  'job_created',
  'job_updated',
  'job_deleted',
];

function subscribeSSE(url, handlers = {}, opts = {}) {
  let es = null;
  let stopped = false;
  let retryMs = 1000;
  let reconnectTimer = null;
  const eventNames = opts.eventNames || DEFAULT_EVENTS;

  const connect = () => {
    if (stopped) return;
    es = new EventSource(url, { withCredentials: true });

    es.onopen = () => {
      retryMs = 1000;
      handlers.onOpen && handlers.onOpen();
    };

    es.onerror = () => {
      handlers.onError && handlers.onError();
      try { es.close(); } catch (_) {}
      if (stopped) return;
      reconnectTimer = setTimeout(connect, retryMs);
      retryMs = Math.min(retryMs * 2, 15000);
    };

    for (const name of eventNames) {
      es.addEventListener(name, (ev) => {
        let data = ev.data;
        try { data = JSON.parse(ev.data); } catch (_) {}
        handlers.onEvent && handlers.onEvent(name, data, ev);
      });
    }

    es.onmessage = (ev) => {
      let data = ev.data;
      try { data = JSON.parse(ev.data); } catch (_) {}
      handlers.onMessage && handlers.onMessage(data);
    };
  };

  connect();
  return () => {
    stopped = true;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    if (es) { try { es.close(); } catch (_) {} }
  };
}

/* ============================ nav ============================ */
document.addEventListener('DOMContentLoaded', () => {
  const path = window.location.pathname;
  document.querySelectorAll('.nav-links a[data-nav]').forEach(a => {
    const href = a.getAttribute('href');
    const active = (href === '/' && path === '/') ||
                   (href !== '/' && path.startsWith(href));
    if (active) a.classList.add('active');
  });

  const logout = document.getElementById('logout-btn');
  if (logout) {
    logout.addEventListener('click', async () => {
      try {
        await api('/logout', { method: 'POST' });
      } catch (_) { /* ignore */ }
      window.location.href = '/login';
    });
  }
});
