const container = document.getElementById('servers-container');
const rowTpl = document.getElementById('server-row-template');

function renumber() {
  [...container.children].forEach((row, i) => {
    row.querySelector('.server-row-title').textContent = `Server ${i + 1}`;
    row.querySelectorAll('input,select').forEach(el => el.name = `${el.dataset.base || el.name}`);
  });
}

function addServer() {
  const node = rowTpl.content.cloneNode(true);
  const row = node.querySelector('.server-row');
  row.querySelector('.remove-server-btn').addEventListener('click', () => {
    row.remove();
    renumber();
  });
  container.appendChild(node);
  renumber();
}

document.getElementById('add-server-btn').addEventListener('click', addServer);
addServer();

document.getElementById('job-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.target;
  const fd = new FormData(form);

  const toNum = (v) => v === '' || v == null ? null : Number(v);
  const rangeFrom = toNum(fd.get('range_from'));
  const rangeTo   = toNum(fd.get('range_to'));

  if (rangeFrom != null && rangeFrom < 0) return toast('From must be ≥ 0', 'error');
  if (rangeTo != null && rangeFrom != null && rangeFrom > rangeTo) {
    return toast('From must be ≤ To', 'error');
  }

  const servers = [...container.children].map(row => {
    const get = (n) => row.querySelector(`[name="${n}"]`)?.value?.trim() || '';
    return {
      name:             get('name'),
      host:             get('host'),
      ssh_port:         Number(get('ssh_port') || 22),
      ssh_user:         get('ssh_user'),
      ssh_key:          get('ssh_key') || null,
      ssh_auth:         get('ssh_auth') || 'key',
      storage_path:     get('storage_path'),
      threads:          get('threads') ? Number(get('threads')) : null,
      min_free_space_gb: get('min_free_space_gb') ? Number(get('min_free_space_gb')) : 0,
    };
  }).filter(s => s.name && s.host && s.ssh_user && s.storage_path);

  if (!servers.length) return toast('Add at least one valid server', 'error');

  const payload = {
    range_from: rangeFrom ?? 0,
    range_to:   rangeTo,          
    request_timeout_seconds: Number(fd.get('request_timeout_seconds') || 20),
    max_retries:             Number(fd.get('max_retries') || 0),
    block_size:              Number(fd.get('block_size') || 5000),
    global_rate_limit_rps:   Number(fd.get('global_rate_limit_rps') || 20),
    stop_on_consecutive_missing:
      fd.get('stop_on_consecutive_missing') === ''
        ? null : Number(fd.get('stop_on_consecutive_missing')),
    threads: Number(fd.get('threads') || 8),
    servers,
  };

  try {
    const created = await api('/jobs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    toast('Job created', 'success');
    const id = created && (created.id || created.job_id);
    setTimeout(() => window.location.href = id ? `/?job=${id}` : '/jobs', 500);
  } catch (err) {
    toast(err.message, 'error');
  }
});
