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
    card.className = 'surface-card soft-cloud-shadow';
    card.style.display = 'flex';
    card.style.flexDirection = 'column';
    card.innerHTML = `
      <div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:16px;">
        <span class="material-symbols-outlined" style="font-size:1.9rem;color:${color};">thermostat</span>
        <button class="rename-btn material-symbols-outlined" data-id="${d.id}" title="Rename" style="font-size:1.2rem;color:var(--on-surface-variant);">edit</button>
      </div>
      <div style="display:flex;align-items:baseline;gap:4px;margin-bottom:8px;">
        <span style="font-family:'Quicksand',sans-serif;font-weight:700;font-size:2rem;color:var(--on-surface);">${c ? c.temperature.toFixed(1) : '–'}</span>
        <span style="color:var(--on-surface-variant);">°C</span>
      </div>
      <div style="display:flex;gap:16px;color:var(--on-surface-variant);font-size:0.9rem;margin-bottom:16px;">
        <span>💧 ${c ? c.humidity.toFixed(0) : '–'}%</span>
        <span>🔋 ${c ? c.battery : '–'}%</span>
      </div>
      <div style="margin-top:auto;padding-top:12px;border-top:1px solid var(--border);">
        <p style="font-family:'Quicksand',sans-serif;font-weight:600;color:var(--on-surface);margin:0;">${escapeHTML(deviceLabel(d))}</p>
        <p style="font-size:0.75rem;color:var(--on-surface-variant);margin:2px 0 0;">Updated: ${updated ? updated.toLocaleString() : 'never'}</p>
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
  const ink = dark ? '#d0c6ab' : '#4d4732';
  const grid = dark ? '#4d4732' : '#d0c6ab';

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

function greetingText() {
  const h = new Date().getHours();
  if (h < 12) return 'Good morning!';
  if (h < 18) return 'Good afternoon!';
  return 'Good evening!';
}

function updateGreeting() {
  document.getElementById('greeting').textContent = greetingText();
  const options = { weekday: 'long', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' };
  document.getElementById('current-datetime').textContent = new Date().toLocaleDateString(undefined, options);
}

function weatherIcon(code) {
  if (code === 0) return 'wb_sunny';
  if (code === 1 || code === 2) return 'partly_cloudy_day';
  if (code === 3) return 'cloud';
  if (code === 45 || code === 48) return 'foggy';
  if ([51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82].includes(code)) return 'rainy';
  if ([71, 73, 75, 77, 85, 86].includes(code)) return 'weather_snowy';
  if ([95, 96, 99].includes(code)) return 'thunderstorm';
  return 'thermostat';
}

function renderWeather(points) {
  const section = document.getElementById('weather-section');
  if (!points.length) {
    section.style.display = 'none';
    return;
  }

  document.getElementById('weather-hourly').innerHTML = points.map((p) => {
    const hour = new Date(p.t).toLocaleTimeString(undefined, { hour: 'numeric' });
    return `
      <div style="display:flex;flex-direction:column;align-items:center;min-width:70px;flex-shrink:0;">
        <p style="font-family:'Quicksand',sans-serif;font-weight:600;font-size:0.85rem;color:var(--on-surface-variant);margin:0 0 8px;">${hour}</p>
        <span class="material-symbols-outlined" style="font-size:1.9rem;color:var(--secondary);margin-bottom:8px;">${weatherIcon(p.weather_code)}</span>
        <p style="font-family:'Quicksand',sans-serif;font-weight:600;font-size:1.1rem;color:var(--on-surface);margin:0;">${Math.round(p.temperature)}°</p>
        <p style="font-size:0.75rem;color:var(--on-surface-variant);margin:4px 0 0;">💧${Math.round(p.precipitation_probability)}% 🌬${Math.round(p.wind_speed)}</p>
      </div>
    `;
  }).join('');

  section.style.display = 'block';
}

async function loadWeather() {
  try {
    const points = await fetchJSON('/api/weather');
    renderWeather(points);
  } catch (e) {
    document.getElementById('weather-section').style.display = 'none';
  }
}

async function init() {
  updateGreeting();
  setInterval(updateGreeting, 60000);

  await loadDevices();
  await loadCurrent();
  await loadHistoryAndRender();
  initRangeButtons();
  connectWS();
  loadWeather();

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    renderCards();
    loadHistoryAndRender();
  });
}

init().catch((err) => {
  console.error(err);
  document.body.insertAdjacentHTML('afterbegin', `<p style="color:#d03b3b">Startup error: ${err.message}</p>`);
});
