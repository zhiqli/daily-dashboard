package handler

import (
	"testing"
	"time"
)

func TestHomeworkAndMenuVisibility(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	store.CreateHomework("数学", "过期作业", yesterday)
	store.CreateHomework("语文", "明日作业", tomorrow)
	homework := store.ListHomework()
	if len(homework) != 1 || homework[0].Content != "明日作业" {
		t.Fatalf("unexpected visible homework: %#v", homework)
	}

	store.CreateMenu("午餐", "昨日菜单", yesterday)
	store.CreateMenu("晚餐", "今日菜单", "")
	menu := store.ListMenu()
	if len(menu) != 1 || menu[0].Content != "今日菜单" {
		t.Fatalf("unexpected visible menu: %#v", menu)
	}
}

func TestHomeworkToggle(t *testing.T) {
	store := NewTodoStore(t.TempDir())
	defer store.db.Close()

	item := store.CreateHomework("英语", "背单词", "")
	done := true
	updated := store.UpdateHomework(item.ID, nil, nil, &done, nil)
	if updated == nil || !updated.Done {
		t.Fatalf("expected completed homework, got %#v", updated)
	}
}
