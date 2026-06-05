const API = {
  weather: '/api/weather',
  quote:   '/api/quote',
  todos:   '/api/todos',
  events:  '/api/events',
};

document.addEventListener('DOMContentLoaded', () => {
  initReaderMode();
  renderCalendar();
  scheduleCalendarRefresh();
  fetchWeather();
  fetchQuote();
  fetchTodos();
  connectSSE();
});

// --- 日期 ---
const LUNAR_INFO = [
  0x04bd8,0x04ae0,0x0a570,0x054d5,0x0d260,0x0d950,0x16554,0x056a0,0x09ad0,0x055d2,
  0x04ae0,0x0a5b6,0x0a4d0,0x0d250,0x1d255,0x0b540,0x0d6a0,0x0ada2,0x095b0,0x14977,
  0x04970,0x0a4b0,0x0b4b5,0x06a50,0x06d40,0x1ab54,0x02b60,0x09570,0x052f2,0x04970,
  0x06566,0x0d4a0,0x0ea50,0x06e95,0x05ad0,0x02b60,0x186e3,0x092e0,0x1c8d7,0x0c950,
  0x0d4a0,0x1d8a6,0x0b550,0x056a0,0x1a5b4,0x025d0,0x092d0,0x0d2b2,0x0a950,0x0b557,
  0x06ca0,0x0b550,0x15355,0x04da0,0x0a5b0,0x14573,0x052b0,0x0a9a8,0x0e950,0x06aa0,
  0x0aea6,0x0ab50,0x04b60,0x0aae4,0x0a570,0x05260,0x0f263,0x0d950,0x05b57,0x056a0,
  0x096d0,0x04dd5,0x04ad0,0x0a4d0,0x0d4d4,0x0d250,0x0d558,0x0b540,0x0b5a0,0x195a6,
  0x095b0,0x049b0,0x0a974,0x0a4b0,0x0b27a,0x06a50,0x06d40,0x0af46,0x0ab60,0x09570,
  0x04af5,0x04970,0x064b0,0x074a3,0x0ea50,0x06b58,0x05ac0,0x0ab60,0x096d5,0x092e0,
  0x0c960,0x0d954,0x0d4a0,0x0da50,0x07552,0x056a0,0x0abb7,0x025d0,0x092d0,0x0cab5,
  0x0a950,0x0b4a0,0x0baa4,0x0ad50,0x055d9,0x04ba0,0x0a5b0,0x15176,0x052b0,0x0a930,
  0x07954,0x06aa0,0x0ad50,0x05b52,0x04b60,0x0a6e6,0x0a4e0,0x0d260,0x0ea65,0x0d530,
  0x05aa0,0x076a3,0x096d0,0x04afb,0x04ad0,0x0a4d0,0x1d0b6,0x0d250,0x0d520,0x0dd45,
  0x0b5a0,0x056d0,0x055b2,0x049b0,0x0a577,0x0a4b0,0x0aa50,0x1b255,0x06d20,0x0ada0,
  0x14b63,0x09370,0x049f8,0x04970,0x064b0,0x168a6,0x0ea50,0x06aa0,0x1a6c4,0x0aae0,
  0x092e0,0x0d2e3,0x0c960,0x0d557,0x0d4a0,0x0da50,0x05d55,0x056a0,0x0a6d0,0x055d4,
  0x052d0,0x0a9b8,0x0a950,0x0b4a0,0x0b6a6,0x0ad50,0x055a0,0x0aba4,0x0a5b0,0x052b0,
  0x0b273,0x06930,0x07337,0x06aa0,0x0ad50,0x14b55,0x04b60,0x0a570,0x054e4,0x0d160,
  0x0e968,0x0d520,0x0daa0,0x16aa6,0x056d0,0x04ae0,0x0a9d4,0x0a2d0,0x0d150,0x0f252
];

function renderCalendar() {
  const now = new Date();
  const beijing = new Date(now.getTime() + 8 * 60 * 60 * 1000);
  const year = beijing.getUTCFullYear();
  const month = beijing.getUTCMonth() + 1;
  const day = beijing.getUTCDate();
  const weekdays = ['星期日','星期一','星期二','星期三','星期四','星期五','星期六'];
  const lunar = solarToLunar(year, month, day);

  document.getElementById('calendar-date').textContent = month + '月' + day + '日';
  document.getElementById('calendar-weekday').textContent = weekdays[beijing.getUTCDay()];
  document.getElementById('calendar-lunar').textContent = lunar;
}

function scheduleCalendarRefresh() {
  const now = new Date();
  const beijing = new Date(now.getTime() + 8 * 60 * 60 * 1000);
  const nextMidnight = Date.UTC(beijing.getUTCFullYear(), beijing.getUTCMonth(), beijing.getUTCDate() + 1);
  const delay = nextMidnight - beijing.getTime() + 1000;

  setTimeout(() => {
    renderCalendar();
    scheduleCalendarRefresh();
  }, delay);
}

function solarToLunar(year, month, day) {
  const baseDate = Date.UTC(1900, 0, 31);
  let offset = Math.floor((Date.UTC(year, month - 1, day) - baseDate) / 86400000);
  let lunarYear = 1900;

  while (lunarYear < 2100 && offset >= lunarYearDays(lunarYear)) {
    offset -= lunarYearDays(lunarYear);
    lunarYear++;
  }

  const leap = leapMonth(lunarYear);
  let lunarMonth = 1;
  let isLeap = false;

  while (lunarMonth <= 12) {
    const days = isLeap ? leapDays(lunarYear) : monthDays(lunarYear, lunarMonth);
    if (offset < days) break;
    offset -= days;
    if (leap === lunarMonth && !isLeap) {
      isLeap = true;
    } else {
      if (isLeap) isLeap = false;
      lunarMonth++;
    }
  }

  return '农历 ' + (isLeap ? '闰' : '') + lunarMonthName(lunarMonth) + lunarDayName(offset + 1);
}

function lunarYearDays(year) {
  let total = 348;
  for (let mask = 0x8000; mask > 0x8; mask >>= 1) {
    if (LUNAR_INFO[year - 1900] & mask) total++;
  }
  return total + leapDays(year);
}

function leapMonth(year) {
  return LUNAR_INFO[year - 1900] & 0xf;
}

function leapDays(year) {
  if (!leapMonth(year)) return 0;
  return (LUNAR_INFO[year - 1900] & 0x10000) ? 30 : 29;
}

function monthDays(year, month) {
  return (LUNAR_INFO[year - 1900] & (0x10000 >> month)) ? 30 : 29;
}

function lunarMonthName(month) {
  return ['正月','二月','三月','四月','五月','六月','七月','八月','九月','十月','冬月','腊月'][month - 1];
}

function lunarDayName(day) {
  const ones = ['一','二','三','四','五','六','七','八','九','十'];
  if (day === 10) return '初十';
  if (day === 20) return '二十';
  if (day === 30) return '三十';
  return day < 10 ? '初' + ones[day - 1] : day < 20 ? '十' + ones[day - 11] : '廿' + ones[day - 21];
}

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
    if (t.due_date) meta += '<span class="todo-meta"><span class="todo-meta-icon todo-meta-icon-clock" aria-hidden="true"></span>' + esc(formatDueDate(t.due_date)) + '</span>';
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

function formatDueDate(value) {
  const parts = value.split('-');
  if (parts.length !== 3) return value;

  const todayParts = beijingDateParts(new Date());
  if (Number(parts[0]) === todayParts.year && Number(parts[1]) === todayParts.month && Number(parts[2]) === todayParts.day) {
    return '今天截止';
  }
  return Number(parts[1]) + '月' + Number(parts[2]) + '日截止';
}

function beijingDateParts(date) {
  const beijing = new Date(date.getTime() + 8 * 60 * 60 * 1000);
  return {
    year: beijing.getUTCFullYear(),
    month: beijing.getUTCMonth() + 1,
    day: beijing.getUTCDate(),
  };
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
      if (ev.type === 'daily_refresh') { renderCalendar(); fetchWeather(); fetchQuote(); fetchTodos(); }
    } catch(err) {}
  };
  es.onerror = () => { es.close(); setTimeout(connectSSE, 5000); };
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}
