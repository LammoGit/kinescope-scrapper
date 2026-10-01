/* global subscribeSSE, api, toast, fmtBytes, fmtNum, timeAgo, escapeHtml */

const listEl = document.getElementById('workers-list');

// Local state: worker_id -> { worker_id, info, metrics }
const workers = new Map();

/* ---------- helpers ---------- */
const STATUS_NAMES = { 0: 'idle', 1: 'working', 2: 'finished' };
const STATUS_CLASSES = { 0: 'badge-idle', 1: 'badge-running', 2: 'badge-completed' };

function statusName(v) {
		return STATUS_NAMES[v] ?? 'unknown';
}
function statusClass(v) {
		return STATUS_CLASSES[v] ?? 'badge-idle';
}

function fmtPct(v) {
		return v == null ? '—' : Number(v).toFixed(1) + '%';
}

function fmtUptime(sec) {
		if (!sec) return '—';
		const d = Math.floor(sec / 86400);
		const h = Math.floor((sec % 86400) / 3600);
		const m = Math.floor((sec % 3600) / 60);
		if (d) return `${d}d ${h}h`;
		if (h) return `${h}h ${m}m`;
		return `${m}m`;
}

/* ---------- render ---------- */
function renderWorker(entry) {
		const info = entry.info || {};
		const m = entry.metrics || {};
		const id = entry.worker_id || info.name || '?';

		const status = statusName(m.status);
		const badgeClass = statusClass(m.status);

		// Disk usage: bytes used / (used + free)
		const used = m.space_used_bytes ?? 0;
		const free = m.free_space_bytes ?? 0;
		const totalBytes = used + free;
		const diskPct = totalBytes ? (used / totalBytes) * 100 : 0;

		// Job progress
		const from = m.from ?? 0;
		const to   = m.to   ?? 0;
		const cur  = m.current ?? 0;
		const jobTotal = to > from ? (to - from + 1) : 0;
		const jobPct = jobTotal ? Math.min(100, (cur / jobTotal) * 100) : 0;

		const active = Array.isArray(m.active_downloads) ? m.active_downloads : [];

		return `
	<div class="card worker-card" data-worker-id="${escapeHtml(id)}">
	  <div class="worker-head">
		<div>
		  <div class="worker-title">${escapeHtml(info.name || id)}</div>
		  <div class="worker-sub">
			${escapeHtml(info.host || '')}:${escapeHtml(String(info.ssh_port || ''))}
			· ${escapeHtml(info.ssh_user || '')}
			· threads ${fmtNum(info.threads || 0)}
		  </div>
		</div>
		<div style="display:flex;gap:8px;align-items:center">
		  <span class="badge ${badgeClass}">${status}</span>
		  <a class="btn btn-ghost" href="/ssh/${encodeURIComponent(id)}">SSH</a>
		</div>
	  </div>

	  <div class="worker-kv">
		<div class="kv"><span class="k">Uptime</span>   <span class="v">${escapeHtml(fmtUptime(m.uptime))}</span></div>
		<div class="kv"><span class="k">CPU</span>      <span class="v">${fmtPct(m.cpu_load)}</span></div>
		<div class="kv"><span class="k">RAM</span>      <span class="v">${fmtPct(m.ram_usage)}</span></div>
		<div class="kv"><span class="k">Net</span>      <span class="v">${fmtPct(m.network_usage)}</span></div>
		<div class="kv"><span class="k">Sys load</span><span class="v">${(m.system_load ?? 0).toFixed(2)}</span></div>
		<div class="kv"><span class="k">Disk</span>     <span class="v">${fmtBytes(used)} / ${fmtBytes(totalBytes)}</span></div>
	  </div>

	  <div class="worker-kv">
		<div class="kv"><span class="k">Checks</span>     <span class="v">${fmtNum(m.active_checks || 0)}</span></div>
		<div class="kv"><span class="k">Downloads</span>  <span class="v">${fmtNum(active.length)}</span></div>
		<div class="kv"><span class="k">Speed</span>      <span class="v">${(m.download_speed ?? 0).toFixed(1)} MB/s</span></div>
		<div class="kv"><span class="k">Found</span>      <span class="v">${fmtNum(m.found || 0)}</span></div>
		<div class="kv"><span class="k">Downloaded</span> <span class="v">${fmtNum(m.downloaded || 0)}</span></div>
		<div class="kv"><span class="k">Errors</span>     <span class="v ${m.errors ? 'text-danger' : ''}">${fmtNum(m.errors || 0)}</span></div>
	  </div>

	  ${jobTotal ? `
		<div class="worker-kv">
		  <div class="kv" style="grid-column: 1 / -1">
			<span class="k">Job [${from}..${to}]</span>
			<span class="v">${fmtNum(cur)} / ${fmtNum(to)} (${jobPct.toFixed(1)}%)</span>
		  </div>
		  <div class="progress-bar" style="grid-column: 1 / -1; margin-top: 4px">
			<div style="width:${jobPct}%"></div>
		  </div>
		</div>` : ''}

	  ${active.length ? `
		<div class="downloads-list">
		  ${active.slice(0, 6).map(url => `
			<div class="dl-row">
			  <div class="dl-title" title="${escapeHtml(url)}">
				<code>${escapeHtml(url.replace(/^https?:\/\//, '').slice(0, 60))}</code>
			  </div>
			</div>`).join('')}
		  ${active.length > 6 ? `<div class="muted small">+${active.length - 6} more…</div>` : ''}
		</div>` : ''}
	</div>`;
}

function renderAll() {
		if (!workers.size) {
				listEl.innerHTML = '<div class="card muted">No workers registered yet.</div>';
				return;
		}
		const entries = [...workers.values()].sort((a, b) =>
				String(a.worker_id).localeCompare(String(b.worker_id))
		);
		listEl.innerHTML = entries.map(renderWorker).join('');
}

/* ---------- merge helpers ---------- */
function upsertWorker(payload) {
		if (!payload || !payload.worker_id) return;
		const existing = workers.get(payload.worker_id) || { worker_id: payload.worker_id };
		workers.set(payload.worker_id, {
				worker_id: payload.worker_id,
				info:      payload.info    ?? existing.info    ?? {},
				metrics:   payload.metrics ?? existing.metrics ?? {},
		});
		renderAll();
}

/* ---------- live stream ---------- */
subscribeSSE('/api/stream/workers', {
		onEvent: (name, payload) => {
				if (name === 'snapshot') {
						workers.clear();
						const raw = payload || {};
						const list = Array.isArray(raw) ? raw : Object.entries(raw).map(([id, v]) => {
								if (v && typeof v === 'object' && ('info' in v || 'metrics' in v)) {
										return { worker_id: id, info: v.info || {}, metrics: v.metrics || {} };
								}
								return { worker_id: id, info: v || {}, metrics: {} };
						});
						for (const entry of list) {
								if (entry && entry.worker_id) workers.set(entry.worker_id, entry);
						}
						renderAll();
						return;
				}
				// All live events carry {worker_id, info, metrics}
				if (name === 'worker_registered' || name === 'worker_metrics' ||
						name === 'worker_status'     || name === 'worker_updated') {
						upsertWorker(payload);
				}
		},
		onError: () => toast('Live worker updates disconnected — retrying', 'warn'),
});

/* Manual refresh button stays as a fallback. */
document.getElementById('workers-refresh')?.addEventListener('click', async () => {
		try {
				const data = await api('/workers');
				workers.clear();
				// REST /workers returns map[string]ServerInfo — reshape to our entry format.
				for (const [id, info] of Object.entries(data || {})) {
						workers.set(id, { worker_id: id, info, metrics: {} });
				}
				renderAll();
		} catch (e) {
				listEl.innerHTML = `<div class="card text-danger">${escapeHtml(e.message)}</div>`;
		}
});
