package model

import "time"

type Todo struct {
	ID        string    `json:"id"`
	Date      string    `json:"date"`
	Content   string    `json:"content"`
	Done      bool      `json:"done"`
	DueDate   string    `json:"due_date"` // YYYY-MM-DD，默认创建当天，展示到截止日期当天
	Assignee  string    `json:"assignee"` // 空表示未设置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Homework struct {
	ID        string    `json:"id"`
	Date      string    `json:"date"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Done      bool      `json:"done"`
	DueDate   string    `json:"due_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MenuItem struct {
	ID        string    `json:"id"`
	Date      string    `json:"date"`
	Meal      string    `json:"meal"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
	Type   string `json:"type"`
}

type WeatherInfo struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	UpdatedAt   string  `json:"updated_at"`
}

type AgentSummaryRequest struct {
	Date    string          `json:"date"`
	Todos   []AgentTodoItem `json:"todos"`
	Summary string          `json:"summary"`
}

type AgentTodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Done     bool   `json:"done"`
	DueDate  string `json:"due_date,omitempty"`
	Assignee string `json:"assignee,omitempty"`
}

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
