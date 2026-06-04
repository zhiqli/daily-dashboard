const API = {
  weather: '/api/weather',
  quote:   '/api/quote',
  todos:   '/api/todos',
  events:  '/api/events',
};

document.addEventListener('DOMContentLoaded', () => {
  initReaderMode();
  fetchWeather();
  fetchQuote();
  fetchTodos();
  connectSSE();
});

// --- 天气 ---
// 心知天气 emoji 映射（文本关键词匹配）
function weatherToEmoji(text) {
  text = text || '';
  if (text.includes('雪')) return '🌨️';
  if (text.includes('雷')) return '⛈️';
  if (text.includes('雨') || text.includes('阵')) return '🌧️';
  if (text.includes('雾') || text.includes('霾') || text.includes('尘')) return '🌫️';
  if (text.includes('云') || text.includes('阴')) return '☁️';
  if (text.includes('晴')) return '☀️';
  return '🌤️';
}

async function fetchWeather() {
  try {
    const w = await (await fetch(API.weather)).json();
    const condition = w.condition || '--';
    document.getElementById('weather-emoji').textContent = weatherToEmoji(condition);
    document.getElementById('weather-location').textContent = w.city || '深圳·宝安';
    document.getElementById('weather-temp').textContent = w.temperature + '°C';
    const meta = [condition];
    if (w.humidity > 0) meta.push('湿度' + w.humidity + '%');
    if (w.wind_speed > 0) meta.push('风速' + w.wind_speed + 'm/s');
    document.getElementById('weather-meta').textContent = meta.join(' · ');
  } catch(e) {}
}

// --- 名言 ---
async function fetchQuote() {
  try {
    const q = await (await fetch(API.quote)).json();
    document.getElementById('quote-text').textContent = '「' + q.text + '」';
    document.getElementById('quote-author').textContent = '—— ' + q.author;
  } catch(e) {}
}

// --- Todo ---
async function fetchTodos() {
  try {
    const todos = await (await fetch(API.todos)).json();
    renderTodos(todos);
  } catch(e) {}
}

function renderTodos(todos) {
  const list = document.getElementById('todo-list');
  if (!todos || todos.length === 0) {
    list.innerHTML = '<li class="empty-state">今日暂无待办</li>';
    renderUpdateTime();
    return;
  }
  list.innerHTML = todos.map(t => {
    let meta = '';
    if (t.due_date) meta += '<span class="todo-meta"><span class="todo-meta-icon todo-meta-icon-clock" aria-hidden="true"></span>' + esc(t.due_date) + '</span>';
    if (t.assignee) meta += '<span class="todo-meta"><span class="todo-meta-icon todo-meta-icon-person" aria-hidden="true"></span>' + esc(t.assignee) + '</span>';
    return `
    <li class="todo-item" data-id="${t.id}">
      <span class="todo-marker${t.done ? ' done' : ''}"></span>
      <span class="todo-content${t.done ? ' done' : ''}">${esc(t.content)}</span>
      ${meta ? '<span class="todo-meta-row">' + meta + '</span>' : ''}
    </li>`;
  }).join('');

  renderUpdateTime();
}

function renderUpdateTime() {
  const ts = formatBeijingTime(new Date());
  let ft = document.getElementById('update-foot');
  if (!ft) {
    ft = document.createElement('div');
    ft.id = 'update-foot';
    ft.className = 'update-time';
    document.getElementById('todos-panel').appendChild(ft);
  }
  ft.textContent = 'updated ' + ts;
}

// --- Kindle 阅读模式 ---
function initReaderMode() {
  const shouldStartReader =
    queryValue('reader') === '1' ||
    queryValue('mode') === 'reader';

  setReaderMode(shouldStartReader);
}

function setReaderMode(enabled) {
  toggleClass(document.documentElement, 'reader-mode', enabled);
  toggleClass(document.body, 'reader-mode', enabled);
}

function formatBeijingTime(date) {
  const beijing = new Date(date.getTime() + 8 * 60 * 60 * 1000);
  return pad2(beijing.getUTCHours()) + ':' + pad2(beijing.getUTCMinutes());
}

function pad2(n) {
  return ('0' + n).slice(-2);
}

function queryValue(name) {
  const query = window.location.search.replace(/^\?/, '').split('&');
  for (let i = 0; i < query.length; i++) {
    const pair = query[i].split('=');
    if (decodeURIComponent(pair[0] || '') === name) {
      return decodeURIComponent(pair[1] || '');
    }
  }
  return '';
}

function toggleClass(el, className, enabled) {
  if (!el || !el.classList) return;
  if (enabled) {
    el.classList.add(className);
  } else {
    el.classList.remove(className);
  }
}

// --- SSE ---
function connectSSE() {
  const es = new EventSource(API.events);
  es.onmessage = (e) => {
    try {
      const ev = JSON.parse(e.data);
      if (['todo_created','todo_updated','todo_deleted','agent_summary'].includes(ev.type)) fetchTodos();
      if (ev.type === 'daily_refresh') { fetchWeather(); fetchQuote(); fetchTodos(); }
    } catch(err) {}
  };
  es.onerror = () => { es.close(); setTimeout(connectSSE, 5000); };
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}
