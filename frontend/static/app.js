const PALETTE = {
  light: ['#2a78d6', '#008300', '#e87ba4', '#eda100', '#1baf7a', '#eb6834', '#4a3aa7', '#e34948'],
  dark: ['#3987e5', '#008300', '#d55181', '#c98500', '#199e70', '#d95926', '#9085e9', '#e66767'],
};

function isDark() {
  const t = document.documentElement.getAttribute('data-theme');
  if (t === 'dark') return true;
  if (t === 'light') return false;
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function colorForIndex(i) {
  const p = isDark() ? PALETTE.dark : PALETTE.light;
  return p[i % p.length];
}

const state = {
  devices: [],
  deviceIndex: new Map(),
  current: new Map(),
  tempChart: null,
  humChart: null,
  range: '24h',
};

async function fetchJSON(url, opts) {
  const res = await fetch(url, opts);
  if (!res.ok) throw new Error(`${url}: ${res.status}`);
  return res.json();
}

function deviceLabel(d) {
  if (d.room) return `${d.name} (${d.room})`;
  return d.name || d.mac;
}

function escapeHTML(s) {
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

async function loadDevices() {
  const devices = await fetchJSON('/api/devices');
  devices.sort((a, b) => a.id - b.id);
  state.devices = devices;
  state.deviceIndex.clear();
  devices.forEach((d, i) => state.deviceIndex.set(d.id, i));
}

async function loadCurrent() {
  const rows = await fetchJSON('/api/current');
  state.current.clear();
  rows.forEach((r) => state.current.set(r.device_id, r));
  renderCards();
}

function renderCards() {
  const container = document.getElementById('cards');
  container.innerHTML = '';

  state.devices.forEach((d) => {
    const c = state.current.get(d.id);
    const color = colorForIndex(state.deviceIndex.get(d.id));
    const updated = c ? new Date(c.updated_at) : null;

    const card = document.createElement('article');
    card.className = 'card';
    card.innerHTML = `
      <div class="card-head">
        <span class="dot" style="background:${color}"></span>
        <button class="rename-btn" data-id="${d.id}" title="Rename">${escapeHTML(deviceLabel(d))} ✎</button>
      </div>
      <div class="card-body">
        <div class="metric"><span class="value">${c ? c.temperature.toFixed(1) : '–'}</span><span class="unit">°C</span></div>
        <div class="sub">💧 ${c ? c.humidity.toFixed(0) : '–'}%　🔋 ${c ? c.battery : '–'}%</div>
        <div class="sub muted">Updated: ${updated ? updated.toLocaleString() : 'never'}</div>
      </div>
    `;
    container.appendChild(card);
  });

  container.querySelectorAll('.rename-btn').forEach((btn) => {
    btn.addEventListener('click', () => renameDevice(Number(btn.dataset.id)));
  });
}

async function renameDevice(id) {
  const device = state.devices.find((d) => d.id === id);
  const name = prompt('Sensor name', device.name);
  if (name === null) return;
  const room = prompt('Room (e.g. outdoor, living room, bedroom)', device.room);
  if (room === null) return;

  const updated = await fetchJSON(`/api/devices/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, room, type: device.type }),
  });

  const idx = state.devices.findIndex((d) => d.id === id);
  state.devices[idx] = updated;
  renderCards();
  updateChartLabels();
}

function buildDatasets(points, key) {
  const byDevice = new Map();
  points.forEach((p) => {
    if (!byDevice.has(p.device_id)) byDevice.set(p.device_id, []);
    byDevice.get(p.device_id).push({ x: new Date(p.t), y: p[key] });
  });

  return state.devices.map((d) => {
    const color = colorForIndex(state.deviceIndex.get(d.id));
    return {
      label: deviceLabel(d),
      data: byDevice.get(d.id) || [],
      borderColor: color,
      backgroundColor: color,
      borderWidth: 2,
      pointRadius: 0,
      pointHoverRadius: 4,
      tension: 0.15,
      spanGaps: true,
    };
  });
}

function chartOptions(unit) {
  const dark = isDark();
  const ink = dark ? '#c3c2b7' : '#52514e';
  const grid = dark ? '#2c2c2a' : '#e1e0d9';

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index', intersect: false },
    plugins: {
      legend: { labels: { color: ink, usePointStyle: true } },
      tooltip: {
        callbacks: {
          label: (ctx) => `${ctx.dataset.label}: ${ctx.parsed.y == null ? '–' : ctx.parsed.y.toFixed(1)}${unit}`,
        },
      },
    },
    scales: {
      x: {
        type: 'time',
        time: { tooltipFormat: 'dd/MM HH:mm' },
        ticks: { color: ink },
        grid: { color: grid },
      },
      y: {
        ticks: { color: ink, callback: (v) => `${v}${unit}` },
        grid: { color: grid },
      },
    },
  };
}

async function loadHistoryAndRender() {
  const points = await fetchJSON(`/api/history?range=${state.range}`);

  if (state.tempChart) state.tempChart.destroy();
  if (state.humChart) state.humChart.destroy();

  state.tempChart = new Chart(document.getElementById('temp-chart'), {
    type: 'line',
    data: { datasets: buildDatasets(points, 'temperature') },
    options: chartOptions('°C'),
  });

  state.humChart = new Chart(document.getElementById('humidity-chart'), {
    type: 'line',
    data: { datasets: buildDatasets(points, 'humidity') },
    options: chartOptions('%'),
  });
}

function updateChartLabels() {
  if (!state.tempChart || !state.humChart) return;
  state.tempChart.data.datasets.forEach((ds, i) => { ds.label = deviceLabel(state.devices[i]); });
  state.humChart.data.datasets.forEach((ds, i) => { ds.label = deviceLabel(state.devices[i]); });
  state.tempChart.update();
  state.humChart.update();
}

function pushLivePoint(update) {
  if (!state.deviceIndex.has(update.device_id)) return;
  const idx = state.deviceIndex.get(update.device_id);
  const t = new Date(update.updated_at);

  if (state.tempChart?.data.datasets[idx]) {
    state.tempChart.data.datasets[idx].data.push({ x: t, y: update.temperature });
    state.tempChart.update('none');
  }
  if (state.humChart?.data.datasets[idx]) {
    state.humChart.data.datasets[idx].data.push({ x: t, y: update.humidity });
    state.humChart.update('none');
  }

  state.current.set(update.device_id, {
    device_id: update.device_id,
    mac: update.mac,
    name: update.name,
    room: update.room,
    temperature: update.temperature,
    humidity: update.humidity,
    battery: update.battery,
    rssi: update.rssi,
    updated_at: update.updated_at,
  });
  renderCards();
}

function connectWS() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}/ws`);
  const statusEl = document.getElementById('conn-status');

  ws.onopen = () => { statusEl.textContent = 'live'; statusEl.className = 'badge ok'; };
  ws.onclose = () => {
    statusEl.textContent = 'disconnected — reconnecting…';
    statusEl.className = 'badge err';
    setTimeout(connectWS, 3000);
  };
  ws.onerror = () => ws.close();
  ws.onmessage = (ev) => {
    try { pushLivePoint(JSON.parse(ev.data)); } catch (e) { console.error(e); }
  };
}

function initRangeButtons() {
  document.querySelectorAll('#range-buttons button').forEach((btn) => {
    btn.addEventListener('click', async () => {
      document.querySelectorAll('#range-buttons button').forEach((b) => b.classList.remove('active'));
      btn.classList.add('active');
      state.range = btn.dataset.range;
      await loadHistoryAndRender();
    });
  });
}

async function init() {
  await loadDevices();
  await loadCurrent();
  await loadHistoryAndRender();
  initRangeButtons();
  connectWS();

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    renderCards();
    loadHistoryAndRender();
  });
}

init().catch((err) => {
  console.error(err);
  document.body.insertAdjacentHTML('afterbegin', `<p style="color:#d03b3b">Startup error: ${err.message}</p>`);
});
