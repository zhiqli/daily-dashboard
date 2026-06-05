package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListIncludesTodosThroughDueDate(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	d := time.Now()
	yesterday := d.AddDate(0, 0, -1).Format("2006-01-02")
	todayDate := d.Format("2006-01-02")
	tomorrow := d.AddDate(0, 0, 1).Format("2006-01-02")

	todayOnly := store.Create("today only", "", "")
	expired := store.Create("expired", yesterday, "")
	dueToday := store.Create("due today", todayDate, "")
	dueTomorrow := store.Create("due tomorrow", tomorrow, "")

	for _, todo := range []*struct {
		id   string
		date string
	}{
		{id: expired.ID, date: yesterday},
		{id: dueToday.ID, date: yesterday},
		{id: dueTomorrow.ID, date: yesterday},
	} {
		if _, err := store.db.Exec("UPDATE todos SET date = ? WHERE id = ?", todo.date, todo.id); err != nil {
			t.Fatal(err)
		}
	}

	got := store.List()
	visible := make(map[string]bool, len(got))
	for _, todo := range got {
		visible[todo.Content] = true
	}

	for _, content := range []string{todayOnly.Content, dueToday.Content, dueTomorrow.Content} {
		if !visible[content] {
			t.Errorf("expected %q to be visible", content)
		}
	}
	if visible[expired.Content] {
		t.Errorf("expected %q to be hidden after its due date", expired.Content)
	}
}

func TestCreateDefaultsDueDateToToday(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	todo := store.Create("default deadline", "", "")
	if todo.DueDate != today() {
		t.Fatalf("expected due date %q, got %q", today(), todo.DueDate)
	}

	stored := store.getByID(todo.ID)
	if stored == nil || stored.DueDate != today() {
		t.Fatalf("expected stored due date %q, got %#v", today(), stored)
	}
}

func TestClearingDueDateDefaultsToCreationDate(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	todo := store.Create("reset deadline", time.Now().AddDate(0, 0, 3).Format("2006-01-02"), "")
	empty := ""
	updated := store.Update(todo.ID, nil, nil, &empty, nil)

	if updated == nil || updated.DueDate != todo.Date {
		t.Fatalf("expected due date %q, got %#v", todo.Date, updated)
	}
}

func TestCreateRejectsInvalidDueDate(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/todos",
		strings.NewReader(`{"content":"write report","due_date":"18:00"}`))
	rec := httptest.NewRecorder()

	MakeTodoHandler(store).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestValidDueDate(t *testing.T) {
	for _, value := range []string{"", "2026-06-05", "2028-02-29"} {
		if !validDueDate(value) {
			t.Errorf("expected %q to be valid", value)
		}
	}
	for _, value := range []string{"18:00", "2026-6-5", "2026-02-30"} {
		if validDueDate(value) {
			t.Errorf("expected %q to be invalid", value)
		}
	}
}
