const API = {
  weather: '/api/weather',
  quote:   '/api/quote',
  todos:   '/api/todos',
  events:  '/api/events',
};

document.addEventListener('DOMContentLoaded', () => {
  fetchWeather();
  fetchQuote();
  fetchTodos();
  connectSSE();
});

// --- 天气 ---
// 心知天气 emoji 映射（文本关键词匹配）
function weatherToEmoji(text) {
  if (text.includes('雪'))   return '🌨️';
  if (text.includes('雷'))   return '⛈️';
  if (text.includes('雨') || text.includes('阵')) return '🌧️';
  if (text.includes('雾') || text.includes('霾') || text.includes('尘')) return '🌫️';
  if (text.includes('云') || text.includes('阴')) return '☁️';
  if (text.includes('晴'))   return '☀️';
  return '🌤️';
}

async function fetchWeather() {
  try {
    const w = await (await fetch(API.weather)).json();
    document.getElementById('weather-emoji').textContent = weatherToEmoji(w.condition) || '🌤️';
    document.getElementById('weather-temp').textContent = w.temperature + '°C';
    document.getElementById('weather-meta').innerHTML =
      (w.condition +
      (w.humidity > 0 ? ' &nbsp;湿度' + w.humidity + '%' : '') +
      (w.wind_speed > 0 ? ' &nbsp;风速' + w.wind_speed + 'm/s' : ''));
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
    return;
  }
  list.innerHTML = todos.map(t => {
    let meta = '';
    if (t.due_date)  meta += '<span class="todo-meta">🕐 ' + esc(t.due_date) + '</span>';
    if (t.assignee)  meta += '<span class="todo-meta">👤 ' + esc(t.assignee) + '</span>';
    return `
    <li class="todo-item" data-id="${t.id}">
      <span class="todo-marker${t.done ? ' done' : ''}"></span>
      <span class="todo-content${t.done ? ' done' : ''}">${esc(t.content)}</span>
      ${meta ? '<span class="todo-meta-row">' + meta + '</span>' : ''}
    </li>`;
  }).join('');

  const now = new Date();
  const ts = ('0'+now.getHours()).slice(-2) + ':' + ('0'+now.getMinutes()).slice(-2);
  let ft = document.getElementById('update-foot');
  if (!ft) {
    ft = document.createElement('div');
    ft.id = 'update-foot';
    ft.className = 'update-time';
    document.getElementById('todos-panel').appendChild(ft);
  }
  ft.textContent = '更新于 ' + ts;
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
