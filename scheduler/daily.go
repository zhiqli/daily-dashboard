package scheduler

import (
	"log"
	"time"

	"daily-dashboard/handler"
)

// StartDailyRefresh 启动每日 6:00 定时刷新
func StartDailyRefresh(store *handler.TodoStore) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			wait := next.Sub(now)

			log.Printf("[scheduler] 下次刷新: %s (等待 %v)", next.Format("2006-01-02 15:04:05"), wait.Round(time.Second))

			timer := time.NewTimer(wait)
			<-timer.C

			log.Println("[scheduler] ✅ 执行每日刷新")
			store.DailyRefresh()
		}
	}()
}
