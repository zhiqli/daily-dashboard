---
name: todo-manager
description: Manage daily dashboard todos, homework, and daily menu items via HTTP API. Trigger when the user asks to add, record, delete, modify, complete, or list tasks/todos/待办/作业/家庭作业/菜单/今日菜单, including natural phrases such as “记录一下数学作业，今天完成” or “今天的菜单是”.
metadata:
  short-description: Manage todos, homework, and menus
---

# Todo Manager

Manage todos on the daily dashboard at `http://localhost:8081`.

## API Base

```
http://localhost:8081
```

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | string | yes | 任务内容 |
| done | bool | no | 是否完成 |
| due_date | string | no | 截止日期 YYYY-MM-DD；默认当天，任务持续展示到截止日期当天 |
| assignee | string | no | 待办人 |

## Content Style

- 创建或改写任务时，将 `content` 写成简短、明确、可执行的任务短语。
- 优先使用“动词 + 对象”，例如“提交周报”“预约体检”“确认发布方案”。
- 删除寒暄、背景说明和重复信息；截止日期与待办人只放在对应字段，不重复写进 `content`。
- 尽量控制在 20 个中文字符以内，但不得删掉用户明确要求的关键范围、对象或结果。
- 用户已给出简练任务名时，直接保留，不做无意义改写。

## Operations

### 1. List todos

```
GET /api/todos
```

### 2. Add todo

```
POST /api/todos
Content-Type: application/json

{"content": "写周报"}
{"content": "写周报", "due_date": "2026-06-08", "assignee": "张三"}
```

### 3. Update todo

```
PUT /api/todos/{id}
Content-Type: application/json

{"done": true}
{"content": "新内容"}
{"due_date": "2026-06-08"}
{"assignee": "李四"}
{"done": true, "assignee": "李四"}
```

### 4. Delete todo

```
DELETE /api/todos/{id}
```

### 5. Batch via agent

```
POST /api/agent/summary
Content-Type: application/json

{
  "date": "2026-06-03",
  "todos": [
    {"id": "abc", "content": "写周报", "done": true, "due_date": "2026-06-08", "assignee": "张三"}
  ],
  "summary": "整体总结"
}
```

## Workflow

1. 添加 → POST /api/todos with content (+ optional due_date, assignee)
2. 完成 → find id via GET /api/todos, then PUT /api/todos/{id} with `{"done":true}`
3. 删除 → find id, then DELETE /api/todos/{id}
4. 修改 → find id, then PUT /api/todos/{id} with new fields
5. 设置截止日期 → PUT /api/todos/{id} with `{"due_date":"2026-06-08"}`
6. 设置待办人 → PUT /api/todos/{id} with `{"assignee":"王五"}`

## Natural Language Rules

- Act directly when the intent and content are clear; do not ask the user to restate information already present.
- Convert relative dates using the user's current timezone, then send `YYYY-MM-DD`.
  - `今天完成`、`今天之前完成` → today
  - `明天完成`、`明天之前完成` → tomorrow
  - `周五之前完成` → the nearest upcoming Friday, including today when today is Friday
  - `下周五之前完成` → Friday of next week
- Remove command phrases and deadline phrases from saved content, such as `记录一下`、`帮我记`、`今天完成`、`周五之前完成`.
- When no deadline is stated, omit `due_date`; the server defaults it to today.

## Homework

- List: `GET /api/homework`
- Add: `POST /api/homework` with `{"subject":"数学","content":"完成练习册第 12 页","due_date":"2026-06-10"}`
- Update: `PUT /api/homework/{id}` with any of `subject`, `content`, `due_date`, or `done`
- Complete/uncomplete: `PUT /api/homework/{id}` with `{"done":true}` or `{"done":false}`
- Delete: `DELETE /api/homework/{id}`
- Keep homework content concise; put the subject only in `subject` and the deadline only in `due_date`.
- Treat phrases containing a school subject plus `作业`、`家庭作业`、`练习` or an explicit request to record homework as homework, not a regular todo.
- Extract the subject into `subject`. Remove the subject, `作业`, command wording, and deadline wording from `content`.
- If nothing remains after extraction, use `做作业` as `content`.
- Examples:
  - `记录一下数学作业，今天完成` → `{"subject":"数学","content":"做作业","due_date":"<today>"}`
  - `语文作业，周五之前完成：背诵古诗两首` → `{"subject":"语文","content":"背诵古诗两首","due_date":"<upcoming Friday>"}`
  - `英语听读课文三遍，明天完成` → `{"subject":"英语","content":"听读课文三遍","due_date":"<tomorrow>"}`

## Daily Menu

- List today's menu: `GET /api/menu`
- Add: `POST /api/menu` with `{"meal":"晚餐","content":"番茄炒蛋","date":"2026-06-09"}`
- Update: `PUT /api/menu/{id}` with any of `meal`, `content`, or `date`
- Delete: `DELETE /api/menu/{id}`
- For `今天的菜单是 ...`, split multiple dishes by newlines, commas, `、`, or semicolons and create one menu item per dish.
- Use an explicitly stated `早餐`、`午餐`、`晚餐` or `加餐` as `meal`. When no meal is stated, use `其他`.
- Remove framing phrases such as `今天的菜单是`、`菜单有` before saving dish names.
- Omit `date` for today's menu; the server defaults it to today.
- Example: `今天的菜单是 番茄炒蛋、清炒时蔬，冬瓜排骨汤` → create three items with `meal:"其他"` and the three concise dish names.
