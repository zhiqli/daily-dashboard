package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"daily-dashboard/model"
)

var (
	localQuotes []model.Quote
	quotesMu    sync.RWMutex
	httpClient  = &http.Client{Timeout: 5 * time.Second}
)

// LoadLocalQuotes 从 JSON 文件加载本地名言（作为 fallback）
func LoadLocalQuotes(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var qs []model.Quote
	if err := json.Unmarshal(data, &qs); err != nil {
		return err
	}
	quotesMu.Lock()
	localQuotes = qs
	quotesMu.Unlock()
	return nil
}

// ListQuotesForLog 返回已加载的名言数量
func ListQuotesForLog() int {
	quotesMu.RLock()
	defer quotesMu.RUnlock()
	return len(localQuotes)
}

// QuoteHandler 优先从在线 API 获取，失败则 fallback 到本地
func QuoteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. 尝试今日诗词 API
	if q, ok := fetchJinrishici(); ok {
		json.NewEncoder(w).Encode(q)
		return
	}

	// 2. 尝试一言 API
	if q, ok := fetchHitokoto(); ok {
		json.NewEncoder(w).Encode(q)
		return
	}

	// 3. fallback 本地
	quotesMu.RLock()
	defer quotesMu.RUnlock()
	if len(localQuotes) == 0 {
		http.Error(w, `{"error":"no quotes available"}`, http.StatusInternalServerError)
		return
	}
	q := localQuotes[rand.Intn(len(localQuotes))]
	json.NewEncoder(w).Encode(q)
}

func fetchJinrishici() (model.Quote, bool) {
	resp, err := httpClient.Get("https://v1.jinrishici.com/all.json")
	if err != nil {
		return model.Quote{}, false
	}
	defer resp.Body.Close()

	var j struct {
		Content  string `json:"content"`
		Origin   string `json:"origin"`
		Author   string `json:"author"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil || j.Content == "" {
		return model.Quote{}, false
	}

	return model.Quote{
		Text:   j.Content,
		Author: j.Author,
		Type:   "poem",
	}, true
}

func fetchHitokoto() (model.Quote, bool) {
	resp, err := httpClient.Get("https://v1.hitokoto.cn/?encode=json&charset=utf-8")
	if err != nil {
		return model.Quote{}, false
	}
	defer resp.Body.Close()

	var j struct {
		Hitokoto string `json:"hitokoto"`
		From     string `json:"from"`
		Type     string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&j); err != nil || j.Hitokoto == "" {
		return model.Quote{}, false
	}

	return model.Quote{
		Text:   j.Hitokoto,
		Author: j.From,
		Type:   "quote",
	}, true
}
