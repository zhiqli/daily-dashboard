---
name: todo-manager
description: Manage daily dashboard todo items via HTTP API. Trigger when the user asks to add, delete, modify, complete, or list tasks/todos/待办/待办项.
metadata:
  short-description: CRUD daily dashboard todos
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
| due_date | string | no | 截止时间 HH:MM |
| assignee | string | no | 待办人 |

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
{"content": "写周报", "due_date": "09:30", "assignee": "张三"}
```

### 3. Update todo

```
PUT /api/todos/{id}
Content-Type: application/json

{"done": true}
{"content": "新内容"}
{"due_date": "16:00"}
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
    {"id": "abc", "content": "写周报", "done": true, "due_date": "09:30", "assignee": "张三"}
  ],
  "summary": "整体总结"
}
```

## Workflow

1. 添加 → POST /api/todos with content (+ optional due_date, assignee)
2. 完成 → find id via GET /api/todos, then PUT /api/todos/{id} with `{"done":true}`
3. 删除 → find id, then DELETE /api/todos/{id}
4. 修改 → find id, then PUT /api/todos/{id} with new fields
5. 设置截止时间 → PUT /api/todos/{id} with `{"due_date":"16:00"}`
6. 设置待办人 → PUT /api/todos/{id} with `{"assignee":"王五"}`
