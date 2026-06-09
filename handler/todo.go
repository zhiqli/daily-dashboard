package handler

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"daily-dashboard/model"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type TodoStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	broadcast chan model.SSEEvent
}

func NewTodoStore(dataDir string) *TodoStore {
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "todos.db")

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("[store] 打开 SQLite 失败: %v", err)
	}
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS todos (
		id          TEXT PRIMARY KEY,
		date        TEXT NOT NULL,
		content     TEXT NOT NULL DEFAULT '',
		done        INTEGER NOT NULL DEFAULT 0,
		due_date    TEXT NOT NULL DEFAULT '',
		assignee    TEXT NOT NULL DEFAULT '',
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("[store] 建表失败: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS homework (
		id TEXT PRIMARY KEY, date TEXT NOT NULL, subject TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '', done INTEGER NOT NULL DEFAULT 0,
		due_date TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS menu_items (
		id TEXT PRIMARY KEY, date TEXT NOT NULL, meal TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("[store] 建作业/菜单表失败: %v", err)
	}

	db.Exec("ALTER TABLE todos ADD COLUMN due_date TEXT NOT NULL DEFAULT ''")
	db.Exec("ALTER TABLE todos ADD COLUMN assignee TEXT NOT NULL DEFAULT ''")
	if _, err := db.Exec("UPDATE todos SET due_date = date WHERE due_date = ''"); err != nil {
		log.Printf("[store] 补全默认截止日期失败: %v", err)
	}

	s := &TodoStore{
		db:        db,
		broadcast: make(chan model.SSEEvent, 100),
	}
	log.Printf("[store] SQLite 就绪，当前展示 %d 条 todo", s.countVisible())
	return s
}

func today() string { return time.Now().Format("2006-01-02") }

func (s *TodoStore) countVisible() int {
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM todos WHERE due_date >= ?", today()).Scan(&n)
	return n
}

func (s *TodoStore) BroadcastChan() chan model.SSEEvent { return s.broadcast }

func (s *TodoStore) List() []*model.Todo {
	d := today()
	rows, err := s.db.Query(`SELECT id, date, content, done, due_date, assignee, created_at, updated_at
		FROM todos
		WHERE due_date >= ?
		ORDER BY done ASC, due_date ASC, created_at ASC`, d)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*model.Todo
	for rows.Next() {
		if t := scanTodo(rows); t != nil {
			result = append(result, t)
		}
	}
	return result
}

func (s *TodoStore) Create(content, dueDate, assignee string) *model.Todo {
	now := time.Now()
	d := today()
	dueDate = defaultDueDate(dueDate, d)
	t := &model.Todo{
		ID:        uuid.New().String(),
		Date:      d,
		Content:   content,
		DueDate:   dueDate,
		Assignee:  assignee,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.Exec(`INSERT INTO todos (id, date, content, done, due_date, assignee, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?, ?, ?)`,
		t.ID, d, content, dueDate, assignee, fmtTime(now), fmtTime(now))
	if err != nil {
		log.Printf("[store] 插入失败: %v", err)
		return nil
	}
	s.broadcastEvent("todo_created", t)
	return t
}

func (s *TodoStore) Update(id string, content *string, done *bool, dueDate *string, assignee *string) *model.Todo {
	now := time.Now()
	existing := s.getByID(id)
	if existing == nil {
		return nil
	}
	clauses := []string{"updated_at = ?"}
	args := []interface{}{fmtTime(now)}
	if content != nil {
		clauses = append(clauses, "content = ?")
		args = append(args, *content)
		existing.Content = *content
	}
	if done != nil {
		v := 0
		if *done {
			v = 1
		}
		clauses = append(clauses, "done = ?")
		args = append(args, v)
		existing.Done = *done
	}
	if dueDate != nil {
		normalizedDueDate := defaultDueDate(*dueDate, existing.Date)
		clauses = append(clauses, "due_date = ?")
		args = append(args, normalizedDueDate)
		existing.DueDate = normalizedDueDate
	}
	if assignee != nil {
		clauses = append(clauses, "assignee = ?")
		args = append(args, *assignee)
		existing.Assignee = *assignee
	}
	existing.UpdatedAt = now
	args = append(args, id)
	_, err := s.db.Exec(`UPDATE todos SET `+strings.Join(clauses, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		log.Printf("[store] 更新失败: %v", err)
		return nil
	}
	s.broadcastEvent("todo_updated", existing)
	return existing
}

func (s *TodoStore) Delete(id string) bool {
	r, _ := s.db.Exec("DELETE FROM todos WHERE id = ?", id)
	n, _ := r.RowsAffected()
	if n == 0 {
		return false
	}
	s.broadcastEvent("todo_deleted", map[string]string{"id": id})
	return true
}

func (s *TodoStore) ApplyAgentSummary(req *model.AgentSummaryRequest) {
	now := time.Now()
	var ids []string
	for _, item := range req.Todos {
		if !validDueDate(item.DueDate) {
			log.Printf("[store] 跳过截止日期无效的 todo: %q", item.DueDate)
			continue
		}
		existing := s.getByID(item.ID)
		if existing == nil {
			t := s.Create(item.Content, item.DueDate, item.Assignee)
			if t != nil {
				item.ID = t.ID
				existing = t
			} else {
				continue
			}
		}
		item.DueDate = defaultDueDate(item.DueDate, existing.Date)
		doneVal := 0
		if item.Done {
			doneVal = 1
		}
		_, err := s.db.Exec(`UPDATE todos SET content=?, done=?, due_date=?, assignee=?, updated_at=? WHERE id=?`,
			item.Content, doneVal, item.DueDate, item.Assignee, fmtTime(now), item.ID)
		if err != nil {
			continue
		}
		ids = append(ids, item.ID)
	}
	s.broadcastEvent("agent_summary", map[string]interface{}{
		"date": req.Date, "summary": req.Summary, "updated_ids": ids,
	})
}

func (s *TodoStore) DailyRefresh() {
	log.Printf("[store] 每日刷新 — %s", today())
	s.broadcastEvent("daily_refresh", map[string]string{
		"date": today(), "time": time.Now().Format("2006-01-02 06:00"),
	})
}

func (s *TodoStore) getByID(id string) *model.Todo {
	row := s.db.QueryRow(`SELECT id, date, content, done, due_date, assignee, created_at, updated_at
		FROM todos WHERE id = ?`, id)
	return scanTodo(row)
}

func scanTodo(scanner interface{ Scan(...interface{}) error }) *model.Todo {
	var t model.Todo
	var doneInt int
	var createdStr, updatedStr string
	err := scanner.Scan(&t.ID, &t.Date, &t.Content, &doneInt, &t.DueDate, &t.Assignee, &createdStr, &updatedStr)
	if err != nil {
		return nil
	}
	t.Done = doneInt != 0
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &t
}

func fmtTime(t time.Time) string { return t.Format(time.RFC3339) }

func validDueDate(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func defaultDueDate(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *TodoStore) broadcastEvent(typ string, data interface{}) {
	select {
	case s.broadcast <- model.SSEEvent{Type: typ, Data: data}:
	default:
		{
		}
	}
}

// --- HTTP Handlers ---

func MakeTodoHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			list := store.List()
			if list == nil {
				list = []*model.Todo{}
			}
			json.NewEncoder(w).Encode(list)
		case http.MethodPost:
			var req struct {
				Content  string `json:"content"`
				DueDate  string `json:"due_date,omitempty"`
				Assignee string `json:"assignee,omitempty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
				http.Error(w, `{"error":"content required"}`, http.StatusBadRequest)
				return
			}
			if !validDueDate(req.DueDate) {
				http.Error(w, `{"error":"due_date must be YYYY-MM-DD"}`, http.StatusBadRequest)
				return
			}
			t := store.Create(req.Content, req.DueDate, req.Assignee)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(t)
		case http.MethodPut:
			id := extractID(r.URL.Path)
			if id == "" {
				http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
				return
			}
			var req struct {
				Content  *string `json:"content,omitempty"`
				Done     *bool   `json:"done,omitempty"`
				DueDate  *string `json:"due_date,omitempty"`
				Assignee *string `json:"assignee,omitempty"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
				return
			}
			if req.DueDate != nil && !validDueDate(*req.DueDate) {
				http.Error(w, `{"error":"due_date must be YYYY-MM-DD"}`, http.StatusBadRequest)
				return
			}
			t := store.Update(id, req.Content, req.Done, req.DueDate, req.Assignee)
			if t == nil {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(t)
		case http.MethodDelete:
			id := extractID(r.URL.Path)
			if id == "" {
				http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
				return
			}
			if !store.Delete(id) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

func MakeAgentSummaryHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var req model.AgentSummaryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		store.ApplyAgentSummary(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func extractID(path string) string {
	const p = "/api/todos/"
	if !strings.HasPrefix(path, p) {
		return ""
	}
	id := strings.TrimPrefix(path, p)
	if id == "" {
		return ""
	}
	return id
}
