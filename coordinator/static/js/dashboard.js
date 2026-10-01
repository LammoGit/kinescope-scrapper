/* global api, subscribeSSE, toast, fmtNum, fmtBytes, fmtSpeed, fmtDuration, statusBadge, escapeHtml, fmtTime */

const state = {
  jobs: [],
  currentJobId: null,
  snapshot: null,
  unsubscribeJob: null,
  unsubscribeEvents: null,
  unsubscribeJobs: null,
};

const $ = (id) => document.getElementById(id);

/* ---------------- jobs list (initial load; live updates via SSE) ---------------- */
async function loadJobs() {
  const data = await api('/jobs');
  state.jobs = Array.isArray(data) ? data : (data.items || []);
  renderJobOptions();
}

function renderJobOptions() {
  const sel = $('job-select');
  const prev = sel.value;
  sel.innerHTML = '';

  if (!state.jobs.length) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = 'No jobs';
    sel.appendChild(opt);
    renderEmpty();
    return;
  }

  for (const j of state.jobs) {
    const opt = document.createElement('option');
    opt.value = j.id;
    opt.textContent = `#${j.id} · ${j.status || 'created'} · [${j.range_from ?? 0}${j.range_to != null ? '…' + j.range_to : '…'}]`;
    sel.appendChild(opt);
  }

  const saved = localStorage.getItem('vd-job');
  const pick = state.currentJobId
    || (saved && state.jobs.find(j => String(j.id) === String(saved)) ? saved : null)
    || state.jobs[0].id;

  sel.value = pick;
  if (String(pick) !== String(prev)) {
    selectJob(pick);
  }
}

$('job-select').addEventListener('change', (e) => {
  localStorage.setItem('vd-job', e.target.value);
  selectJob(e.target.value);
});

/* ---------------- job selection: tear down old SSE, open new ---------------- */
function selectJob(jobId) {
  if (state.unsubscribeJob)    { state.unsubscribeJob();    state.unsubscribeJob = null; }
  if (state.unsubscribeEvents) { state.unsubscribeEvents(); state.unsubscribeEvents = null; }

  state.currentJobId = jobId;
  if (!jobId) return renderEmpty();

  state.unsubscribeJob = subscribeSSE(`/api/stream/jobs/${encodeURIComponent(jobId)}`, {
    onEvent: (name, payload) => {
      if (name === 'snapshot' || name === 'progress') renderJob(payload);
    },
    onError: () => toast('Live job updates disconnected — retrying', 'warn'),
  });

  state.unsubscribeEvents = subscribeSSE(`/api/stream/jobs/${encodeURIComponent(jobId)}/events`, {
    onEvent: (name, payload) => {
      if (name === 'snapshot') {
        renderEvents(Array.isArray(payload) ? payload : []);
      } else if (name === 'event') {
        prependEvent(payload);
      }
    },
  });
}

/* ---------------- job actions ---------------- */
async function jobAction(action) {
  if (!state.currentJobId) return;
  try {
    await api(`/jobs/${state.currentJobId}/${action}`, { method: 'POST' });
    toast(`Job ${action}`, 'success');
    // no manual refresh — server publishes "jobs" and "job:<id>" snapshots
  } catch (e) {
    toast(e.message, 'error');
  }
}
$('btn-start').addEventListener('click',  () => jobAction('start'));
$('btn-pause').addEventListener('click',  () => jobAction('pause'));
$('btn-resume').addEventListener('click', () => jobAction('resume'));
$('btn-stop').addEventListener('click',   () => {
  if (confirm('Stop this job? Active operations will be cancelled (checkpoints kept).')) {
    jobAction('stop');
  }
});

/* ---------------- render (unchanged) ---------------- */
function renderEmpty() {
  $('job-title').textContent = 'No job selected';
  $('job-range').textContent = '—';
  $('job-status').textContent = 'idle';
  $('job-status').className = 'badge badge-idle';
  $('progress-bar').style.width = '0%';
  $('progress-text').textContent = '0 / 0';
  $('progress-pct').textContent = '0%';
  ['stat-processed','stat-found','stat-downloaded','stat-errors',
   'speed-process','speed-download','total-data','eta',
   'active-servers','active-threads'].forEach(id => $(id).textContent = '0');
  $('events-list').innerHTML = '<div class="muted small">No events yet.</div>';
}

function renderJob(job) {
  if (!job) return renderEmpty();

  $('job-title').textContent = `Job #${job.id}`;
  const from = job.range_from ?? 0;
  const to   = (job.range_to == null) ? '∞' : job.range_to;
  $('job-range').textContent = `Range: [${from}, ${to}] · ${job.status || 'created'}`;
  $('job-status').textContent = job.status || 'idle';
  $('job-status').className = 'badge ' + (statusBadge(job.status).match(/badge (badge-\w+)/)?.[1] || 'badge-idle');

  const total     = job.total_count    ?? (to === '∞' ? null : (to - from + 1));
  const processed = job.processed_count ?? 0;
  const pct = total ? Math.min(100, (processed / total) * 100) : 0;
  $('progress-bar').style.width = `${pct.toFixed(2)}%`;
  $('progress-text').textContent = `${fmtNum(processed)} / ${total == null ? '∞' : fmtNum(total)}`;
  $('progress-pct').textContent = total ? `${pct.toFixed(1)}%` : '—';

  $('stat-processed').textContent  = fmtNum(processed);
  $('stat-found').textContent      = fmtNum(job.found_count);
  $('stat-downloaded').textContent = fmtNum(job.downloaded_count);
  $('stat-errors').textContent     = fmtNum(job.error_count);

  const curCheck = job.check_rate_per_sec      ?? job.current_check_rate  ?? 0;
  const avgCheck = job.avg_check_rate_per_sec  ?? 0;
  $('speed-process').innerHTML =
    `${fmtNum(curCheck)} / ${fmtNum(avgCheck)} <span class="unit">id/s</span>`;

  const curDl = job.download_rate_bps      ?? 0;
  const avgDl = job.avg_download_rate_bps  ?? 0;
  $('speed-download').innerHTML = `${fmtSpeed(curDl)} / ${fmtSpeed(avgDl)}`;

  $('total-data').textContent = fmtBytes(job.total_bytes);

  if (to === '∞') {
    $('eta').textContent = 'undetermined';
  } else if (avgCheck > 0 && total) {
    const remaining = Math.max(0, total - processed);
    $('eta').textContent = fmtDuration(remaining / avgCheck);
  } else {
    $('eta').textContent = '—';
  }

  $('active-servers').textContent = fmtNum(job.active_workers);
  $('active-threads').textContent = fmtNum(job.active_threads);

  const s = (job.status || '').toLowerCase();
  const en  = (b) => b.disabled = false;
  const dis = (b) => b.disabled = true;
  en($('btn-start')); en($('btn-pause')); en($('btn-resume')); en($('btn-stop'));
  if (s === 'running') { dis($('btn-start')); dis($('btn-resume')); }
  if (s === 'paused')  { dis($('btn-start')); dis($('btn-pause'));  }
  if (s === 'completed' || s === 'failed' || s === 'stopped') {
    dis($('btn-start')); dis($('btn-pause')); dis($('btn-resume')); dis($('btn-stop'));
  }
}

function eventRowHtml(ev) {
  return `
    <div class="event-row">
      <span class="ts">${escapeHtml(fmtTime(ev.time))}</span>
      <span class="lvl ${escapeHtml(ev.level || 'info')}">${escapeHtml(ev.level || 'info')}</span>
      <span class="cmp">${escapeHtml(ev.component || '')}</span>
      <span>${escapeHtml(ev.message || '')}</span>
    </div>`;
}

function renderEvents(events) {
  const box = $('events-list');
  if (!events || !events.length) {
    box.innerHTML = '<div class="muted small">No events yet.</div>';
    return;
  }
  box.innerHTML = events.slice(0, 40).map(eventRowHtml).join('');
}

function prependEvent(ev) {
  if (!ev) return;
  const box = $('events-list');
  if (box.firstElementChild && box.firstElementChild.classList.contains('muted')) {
    box.innerHTML = '';
  }
  const tpl = document.createElement('template');
  tpl.innerHTML = eventRowHtml(ev).trim();
  box.prepend(tpl.content.firstChild);
  while (box.children.length > 40) box.removeChild(box.lastElementChild);
}

/* ---------------- init ---------------- */
(async function init() {
  try {
    await loadJobs();
    // If nothing was pre-selected, keep the dashboard in an empty state until
    // the jobs SSE pushes a list.
    if (state.jobs.length && !state.currentJobId) selectJob(state.jobs[0].id);

    // Live-update the job dropdown itself.
    state.unsubscribeJobs = subscribeSSE('/api/stream/jobs', {
      onEvent: (name, payload) => {
        if (name === 'snapshot') {
          state.jobs = Array.isArray(payload) ? payload : (payload.items || []);
          renderJobOptions();
        }
      },
    });
  } catch (e) {
    toast(e.message, 'error');
  }
})();
