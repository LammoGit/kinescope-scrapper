/* global subscribeSSE, api, toast, fmtNum, statusBadge, escapeHtml */

const tbody = document.getElementById('jobs-tbody');

// Local state — same pattern as workers.js so live events can patch in place.
const jobs = new Map();

/* ---------------- render ---------------- */

function renderRow(j) {
  const id  = j.id ?? j.job_id ?? '?';
  const from = j.range_from ?? 0;
  const to   = j.range_to == null ? '∞' : j.range_to;
  const total     = j.total_count    ?? (to === '∞' ? null : (to - from + 1));
  const processed = j.processed_count ?? j.current ?? 0;
  const pct = total ? Math.min(100, (processed / total) * 100).toFixed(1) : '—';

  return `
    <tr data-id="${escapeHtml(id)}">
      <td><a href="/?job=${encodeURIComponent(id)}">#${escapeHtml(id)}</a></td>
      <td><code>[${escapeHtml(from)}, ${escapeHtml(to)}]</code></td>
      <td>${statusBadge(j.status)}</td>
      <td>${pct === '—' ? '—' : pct + '%'} <span class="muted small">(${fmtNum(processed)})</span></td>
      <td class="text-info">${fmtNum(j.found_count)}</td>
      <td class="text-success">${fmtNum(j.downloaded_count)}</td>
      <td class="${j.error_count ? 'text-danger' : ''}">${fmtNum(j.error_count)}</td>
      <td>${fmtNum(j.worker_count ?? j.active_workers ?? 0)}</td>
      <td>
        <div class="row-actions">
          <button class="btn btn-ghost" data-action="start"   data-id="${escapeHtml(id)}">Start</button>
          <button class="btn btn-ghost" data-action="pause"   data-id="${escapeHtml(id)}">Pause</button>
          <button class="btn btn-ghost" data-action="resume"  data-id="${escapeHtml(id)}">Resume</button>
          <button class="btn btn-danger" data-action="stop"   data-id="${escapeHtml(id)}">Stop</button>
        </div>
      </td>
    </tr>`;
}

function renderAll() {
  if (!jobs.size) {
    tbody.innerHTML = `<tr><td colspan="9" class="muted">No jobs yet. <a href="/jobs/create">Create one</a>.</td></tr>`;
    return;
  }
  const list = [...jobs.values()].sort((a, b) =>
    String(a.id ?? a.job_id).localeCompare(String(b.id ?? b.job_id))
  );
  tbody.innerHTML = list.map(renderRow).join('');
}

/* ---------------- state helpers ---------------- */

// Normalize the many possible snapshot shapes into an array of job objects
// that always have an `id` field.
function normalizeSnapshot(payload) {
  if (!payload) return [];

  // Array of objects — server's preferred shape.
  if (Array.isArray(payload)) {
    return payload
      .filter(x => x && typeof x === 'object')
      .map(x => ({ ...x, id: x.id ?? x.job_id }));
  }

  // {items: [...]}
  if (Array.isArray(payload.items)) {
    return payload.items
      .filter(x => x && typeof x === 'object')
      .map(x => ({ ...x, id: x.id ?? x.job_id }));
  }

  // Map {jobID: jobObject}
  if (typeof payload === 'object') {
    return Object.entries(payload)
      .filter(([, v]) => v && typeof v === 'object')
      .map(([id, v]) => ({ ...v, id: v.id ?? v.job_id ?? id }));
  }

  return [];
}

function replaceAll(list) {
  jobs.clear();
  for (const j of list) {
    if (j && j.id != null) jobs.set(String(j.id), j);
  }
  renderAll();
}

function upsert(j) {
  if (!j) return;
  const id = j.id ?? j.job_id;
  if (id == null) return;
  const existing = jobs.get(String(id)) || {};
  jobs.set(String(id), { ...existing, ...j, id });
  renderAll();
}

function remove(id) {
  if (id == null) return;
  jobs.delete(String(id));
  renderAll();
}

/* ---------------- row actions ---------------- */

tbody.addEventListener('click', async (e) => {
  const btn = e.target.closest('button[data-action]');
  if (!btn) return;
  const { action, id } = btn.dataset;
  if (action === 'stop' && !confirm(`Stop job #${id}?`)) return;

  btn.disabled = true;
  try {
    await api(`/jobs/${encodeURIComponent(id)}/${action}`, { method: 'POST' });
    toast(`Job ${id}: ${action}`, 'success');
    // The server pushes job_updated on the SSE stream; no manual refresh.
  } catch (err) {
    toast(err.message, 'error');
  } finally {
    btn.disabled = false;
  }
});

/* ---------------- live stream ---------------- */

let gotSnapshot = false;

const unsubscribe = subscribeSSE('/api/stream/jobs', {
  onEvent: (name, payload) => {
    switch (name) {
      case 'snapshot':
        gotSnapshot = true;
        replaceAll(normalizeSnapshot(payload));
        break;

      case 'job_created':
      case 'job_updated':
        // Either a full job object, or just {id, ...changedFields}.
        upsert(payload);
        break;

      case 'job_deleted':
        remove(payload && (payload.id ?? payload.job_id ?? payload));
        break;

      case 'error':
        toast((payload && payload.error) || 'Stream error', 'error');
        break;
    }
  },
  onError: () => {
    // Don't spam on transient reconnects — only warn if we never got anything.
    if (!gotSnapshot) toast('Live job list disconnected — retrying', 'warn');
  },
});

/* ---------------- fallback: REST fetch ---------------- */

// If the SSE connection is slow to establish, fetch once so the table isn't
// stuck on "Loading…". Guarded by `gotSnapshot` so it doesn't stomp on live data.
(async function initialFetch() {
  try {
    const data = await api('/jobs');
    if (!gotSnapshot) {
      replaceAll(normalizeSnapshot(data));
    }
  } catch (e) {
    if (!gotSnapshot) {
      tbody.innerHTML = `<tr><td colspan="9" class="text-danger">${escapeHtml(e.message)}</td></tr>`;
    }
  }
})();

// Cleanup on navigation (matters for SPA-style routers; harmless otherwise).
window.addEventListener('beforeunload', () => {
  if (typeof unsubscribe === 'function') unsubscribe();
});
