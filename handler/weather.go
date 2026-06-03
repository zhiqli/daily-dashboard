package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"daily-dashboard/model"
)

const (
	seniverseAPI = "https://api.seniverse.com/v3/weather/now.json"
	seniverseKey = "Sm_fqJiDSMafsMsFT"
	seniverseLoc = "shenzhen"
	cacheTTL     = 30 * time.Minute
)

var (
	cachedWeather     *model.WeatherInfo
	cachedWeatherTime time.Time
	weatherMu         sync.Mutex
	weatherClient     = &http.Client{Timeout: 8 * time.Second}
)

// WeatherHandler 从心知天气获取深圳实时天气
func WeatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 检查30分钟缓存
	weatherMu.Lock()
	if cachedWeather != nil && time.Since(cachedWeatherTime) < cacheTTL {
		info := *cachedWeather
		weatherMu.Unlock()
		json.NewEncoder(w).Encode(info)
		return
	}
	weatherMu.Unlock()

	info, err := fetchSeniverseWeather()
	if err != nil {
		// 有旧缓存则降级返回
		weatherMu.Lock()
		if cachedWeather != nil {
			fallback := *cachedWeather
			weatherMu.Unlock()
			json.NewEncoder(w).Encode(fallback)
			return
		}
		weatherMu.Unlock()
		http.Error(w, `{"error":"天气服务暂时不可用"}`, http.StatusServiceUnavailable)
		return
	}

	weatherMu.Lock()
	cachedWeather = &info
	cachedWeatherTime = time.Now()
	weatherMu.Unlock()

	json.NewEncoder(w).Encode(info)
}

func fetchSeniverseWeather() (model.WeatherInfo, error) {
	url := seniverseAPI + "?key=" + seniverseKey + "&location=" + seniverseLoc + "&language=zh-Hans&unit=c"
	resp, err := weatherClient.Get(url)
	if err != nil {
		return model.WeatherInfo{}, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Location struct {
				Name string `json:"name"`
			} `json:"location"`
			Now struct {
				Text        string `json:"text"`
				Temperature string `json:"temperature"`
				Humidity    string `json:"humidity"`
				WindSpeed   string `json:"wind_speed"`
			} `json:"now"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return model.WeatherInfo{}, err
	}

	if len(result.Results) == 0 {
		return model.WeatherInfo{}, fmt.Errorf("empty results")
	}

	r := result.Results[0]
	temp, _ := strconv.ParseFloat(r.Now.Temperature, 64)
	humidity, _ := strconv.Atoi(r.Now.Humidity)
	wind, _ := strconv.ParseFloat(r.Now.WindSpeed, 64)

	return model.WeatherInfo{
		City:        "深圳·宝安",
		Temperature: temp,
		Condition:   r.Now.Text,
		Humidity:    humidity,
		WindSpeed:   wind,
		UpdatedAt:   time.Now().Format("15:04"),
	}, nil
}
