package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"daily-dashboard/model"
)

// SSEBroker 管理 SSE 客户端连接
type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewSSEBroker 创建 SSE 广播器
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[chan string]struct{}),
	}
}

// AddClient 添加客户端
func (b *SSEBroker) AddClient(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[ch] = struct{}{}
}

// RemoveClient 移除客户端
func (b *SSEBroker) RemoveClient(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

// Broadcast 向所有客户端广播
func (b *SSEBroker) Broadcast(event model.SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[sse] marshal error: %v", err)
		return
	}
	msg := fmt.Sprintf("data: %s\n\n", data)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// 客户端消费慢，跳过
		}
	}
}

// StartBroadcastLoop 从 TodoStore 的广播通道消费并推送给 SSE 客户端
func (b *SSEBroker) StartBroadcastLoop(eventCh chan model.SSEEvent) {
	for event := range eventCh {
		b.Broadcast(event)
	}
}

// SSEHandler 处理 SSE 连接
func SSEHandler(broker *SSEBroker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := make(chan string, 50)
		broker.AddClient(ch)

		// 连接断开时清理
		defer broker.RemoveClient(ch)

		ctx := r.Context()
		for {
			select {
			case msg := <-ch:
				fmt.Fprint(w, msg)
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	}
}
