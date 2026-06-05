# 每日面板 (Daily Dashboard)

极简风格的墨水屏 / 窄屏每日仪表盘。启动后显示天气、名言、今日待办三张卡片，每天凌晨 6:00 自动刷新。

## 功能

| 卡片 | 数据来源 | 说明 |
|------|----------|------|
| 天气 | 心知天气 API | 深圳实时温度、天气状况，30 分钟缓存 |
| 名言 | 今日诗词 → 一言 → 本地 fallback | 三级降级，保证始终有内容 |
| 待办 | SQLite 本地持久化 | 支持 CRUD、SSE 实时推送、AI Agent 批量同步 |

## 快速开始

```bash
# 编译
go build -o daily-dashboard .

# 启动（默认 8081，可通过 PORT 环境变量指定）
PORT=8081 ./daily-dashboard
```

浏览器打开 `http://localhost:8081`。

Kindle 上可打开 `http://<设备可访问地址>:8081/?reader=1` 直接进入电子书阅读模式。

## Todo 操作

```bash
# 添加
curl -X POST localhost:8081/api/todos \
  -H 'Content-Type: application/json' \
  -d '{"content":"写周报","due_date":"2026-06-08","assignee":"张三"}'

# 列表
curl localhost:8081/api/todos

# 完成
curl -X PUT localhost:8081/api/todos/{id} -d '{"done":true}'

# 删除
curl -X DELETE localhost:8081/api/todos/{id}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | 任务内容 |
| done | bool | 否 | 是否完成 |
| due_date | string | 否 | 截止日期 `YYYY-MM-DD`；默认当天，并持续展示到截止日期当天 |
| assignee | string | 否 | 待办人 |

## API 一览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 主页面 |
| `/api/weather` | GET | 天气数据 |
| `/api/quote` | GET | 每日名言 |
| `/api/todos` | GET / POST | Todo 列表 / 新增 |
| `/api/todos/{id}` | PUT / DELETE | 更新 / 删除 Todo |
| `/api/events` | GET | SSE 事件流 |
| `/api/agent/summary` | POST | AI Agent 批量同步 |
| `/api/refresh` | POST | 手动触发每日刷新 |

## 项目结构

```
daily-dashboard/
├── main.go              # 入口，路由注册
├── handler/
│   ├── weather.go       # 心知天气 API
│   ├── quote.go         # 三级 fallback 名言
│   ├── todo.go          # Todo CRUD + SQLite
│   └── sse.go           # SSE 事件广播
├── model/todo.go        # 数据结构
├── scheduler/daily.go   # 每日 6:00 自动刷新
├── templates/index.html # 页面模板
├── static/
│   ├── style.css        # 墨水屏样式
│   └── app.js           # 前端交互
├── data/
│   ├── todos.db         # SQLite 数据库
│   └── quotes.json      # 本地名言（fallback）
└── go.mod
```

## 技术栈

- **Go 1.22** — HTTP server，`//go:embed` 静态文件
- **SQLite** (`go-sqlite3`) — Todo 持久化
- **心知天气** — 实时气象
- **SSE** — 前端实时推送

## 设计

- 600px 宽度，适配墨水屏 / 竖屏平板
- Todo 单行布局：内容左对齐，时间与待办人右对齐
- 无 JS 框架，原生 JavaScript
- 所有静态资源编译进单一二进制
