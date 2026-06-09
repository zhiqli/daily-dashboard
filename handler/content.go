package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"daily-dashboard/model"

	"github.com/google/uuid"
)

func (s *TodoStore) ListHomework() []*model.Homework {
	rows, err := s.db.Query(`SELECT id, date, subject, content, done, due_date, created_at, updated_at
		FROM homework WHERE due_date >= ? ORDER BY done ASC, due_date ASC, subject ASC, created_at ASC`, today())
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*model.Homework
	for rows.Next() {
		var h model.Homework
		var done int
		var created, updated string
		if rows.Scan(&h.ID, &h.Date, &h.Subject, &h.Content, &done, &h.DueDate, &created, &updated) == nil {
			h.Done = done != 0
			h.CreatedAt, _ = time.Parse(time.RFC3339, created)
			h.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
			result = append(result, &h)
		}
	}
	return result
}

func (s *TodoStore) CreateHomework(subject, content, dueDate string) *model.Homework {
	now := time.Now()
	d := today()
	dueDate = defaultDueDate(dueDate, d)
	h := &model.Homework{ID: uuid.New().String(), Date: d, Subject: subject, Content: content, DueDate: dueDate, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.Exec(`INSERT INTO homework (id,date,subject,content,done,due_date,created_at,updated_at)
		VALUES (?,?,?,?,0,?,?,?)`, h.ID, d, subject, content, dueDate, fmtTime(now), fmtTime(now)); err != nil {
		return nil
	}
	s.broadcastEvent("homework_created", h)
	return h
}

func (s *TodoStore) UpdateHomework(id string, subject, content *string, done *bool, dueDate *string) *model.Homework {
	clauses := []string{"updated_at=?"}
	args := []interface{}{fmtTime(time.Now())}
	if subject != nil {
		clauses = append(clauses, "subject=?")
		args = append(args, *subject)
	}
	if content != nil {
		clauses = append(clauses, "content=?")
		args = append(args, *content)
	}
	if dueDate != nil {
		clauses = append(clauses, "due_date=?")
		args = append(args, defaultDueDate(*dueDate, today()))
	}
	if done != nil {
		value := 0
		if *done {
			value = 1
		}
		clauses = append(clauses, "done=?")
		args = append(args, value)
	}
	if len(clauses) == 1 {
		return nil
	}
	args = append(args, id)
	if result, err := s.db.Exec("UPDATE homework SET "+strings.Join(clauses, ",")+" WHERE id=?", args...); err != nil {
		return nil
	} else if n, _ := result.RowsAffected(); n == 0 {
		return nil
	}
	for _, h := range s.ListHomework() {
		if h.ID == id {
			s.broadcastEvent("homework_updated", h)
			return h
		}
	}
	return nil
}

func (s *TodoStore) DeleteHomework(id string) bool {
	result, _ := s.db.Exec("DELETE FROM homework WHERE id=?", id)
	n, _ := result.RowsAffected()
	if n > 0 {
		s.broadcastEvent("homework_deleted", map[string]string{"id": id})
	}
	return n > 0
}

func (s *TodoStore) ListMenu() []*model.MenuItem {
	rows, err := s.db.Query(`SELECT id, date, meal, content, created_at, updated_at
		FROM menu_items WHERE date = ? ORDER BY CASE meal WHEN '早餐' THEN 1 WHEN '午餐' THEN 2 WHEN '晚餐' THEN 3 ELSE 4 END, created_at ASC`, today())
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*model.MenuItem
	for rows.Next() {
		var item model.MenuItem
		var created, updated string
		if rows.Scan(&item.ID, &item.Date, &item.Meal, &item.Content, &created, &updated) == nil {
			item.CreatedAt, _ = time.Parse(time.RFC3339, created)
			item.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
			result = append(result, &item)
		}
	}
	return result
}

func (s *TodoStore) CreateMenu(meal, content, date string) *model.MenuItem {
	now := time.Now()
	date = defaultDueDate(date, today())
	item := &model.MenuItem{ID: uuid.New().String(), Date: date, Meal: meal, Content: content, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.Exec(`INSERT INTO menu_items (id,date,meal,content,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		item.ID, date, meal, content, fmtTime(now), fmtTime(now)); err != nil {
		return nil
	}
	s.broadcastEvent("menu_created", item)
	return item
}

func (s *TodoStore) DeleteMenu(id string) bool {
	result, _ := s.db.Exec("DELETE FROM menu_items WHERE id=?", id)
	n, _ := result.RowsAffected()
	if n > 0 {
		s.broadcastEvent("menu_deleted", map[string]string{"id": id})
	}
	return n > 0
}

func (s *TodoStore) UpdateMenu(id string, meal, content, date *string) *model.MenuItem {
	clauses := []string{"updated_at=?"}
	args := []interface{}{fmtTime(time.Now())}
	for column, value := range map[string]*string{"meal": meal, "content": content, "date": date} {
		if value != nil {
			clauses = append(clauses, column+"=?")
			args = append(args, *value)
		}
	}
	if len(clauses) == 1 {
		return nil
	}
	args = append(args, id)
	result, err := s.db.Exec("UPDATE menu_items SET "+strings.Join(clauses, ",")+" WHERE id=?", args...)
	if err != nil {
		return nil
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil
	}
	row := s.db.QueryRow("SELECT id,date,meal,content,created_at,updated_at FROM menu_items WHERE id=?", id)
	var item model.MenuItem
	var created, updated string
	if row.Scan(&item.ID, &item.Date, &item.Meal, &item.Content, &created, &updated) != nil {
		return nil
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, created)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	s.broadcastEvent("menu_updated", &item)
	return &item
}

func MakeHomeworkHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/api/homework/")
		switch r.Method {
		case http.MethodGet:
			list := store.ListHomework()
			if list == nil {
				list = []*model.Homework{}
			}
			json.NewEncoder(w).Encode(list)
		case http.MethodPost:
			var req struct {
				Subject string `json:"subject"`
				Content string `json:"content"`
				DueDate string `json:"due_date"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil || req.Subject == "" || req.Content == "" || !validDueDate(req.DueDate) {
				http.Error(w, `{"error":"subject, content and valid due_date required"}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(store.CreateHomework(req.Subject, req.Content, req.DueDate))
		case http.MethodPut:
			var req struct {
				Subject *string `json:"subject"`
				Content *string `json:"content"`
				Done    *bool   `json:"done"`
				DueDate *string `json:"due_date"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil || id == "" || (req.DueDate != nil && !validDueDate(*req.DueDate)) {
				http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
				return
			}
			item := store.UpdateHomework(id, req.Subject, req.Content, req.Done, req.DueDate)
			if item == nil {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(item)
		case http.MethodDelete:
			if !store.DeleteHomework(id) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

func MakeMenuHandler(store *TodoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/api/menu/")
		switch r.Method {
		case http.MethodGet:
			list := store.ListMenu()
			if list == nil {
				list = []*model.MenuItem{}
			}
			json.NewEncoder(w).Encode(list)
		case http.MethodPost:
			var req struct {
				Meal    string `json:"meal"`
				Content string `json:"content"`
				Date    string `json:"date"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil || req.Meal == "" || req.Content == "" || !validDueDate(req.Date) {
				http.Error(w, `{"error":"meal, content and valid date required"}`, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(store.CreateMenu(req.Meal, req.Content, req.Date))
		case http.MethodPut:
			var req struct {
				Meal    *string `json:"meal"`
				Content *string `json:"content"`
				Date    *string `json:"date"`
			}
			if json.NewDecoder(r.Body).Decode(&req) != nil || id == "" || (req.Date != nil && !validDueDate(*req.Date)) {
				http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
				return
			}
			item := store.UpdateMenu(id, req.Meal, req.Content, req.Date)
			if item == nil {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(item)
		case http.MethodDelete:
			if !store.DeleteMenu(id) {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}
