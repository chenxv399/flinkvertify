package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Task 结构体
type Task struct {
	ID        string    `gorm:"primaryKey"`
	URL       string    `json:"url"`
	Status    string    `json:"status"` // processing, success, failure
	Result    bool      `json:"result"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 全局变量
var (
	db       *gorm.DB
	taskMu   sync.Mutex // 保护任务状态更新的锁
	apiKey   string     // API密钥
	keyword1 string     //关键词1
	keyword2 string     //关键词2
)

// 初始化数据库
func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("tasks.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}
	if err = db.AutoMigrate(&Task{}); err != nil {
		log.Fatalf("Failed to migrate database: %s", err)
	}
}

// 初始化随机API密钥
func generateAPIKey() string {
	key := make([]byte, 16)
	rand.Seed(time.Now().UnixNano())
	for i := range key {
		key[i] = byte(rand.Intn(26) + 65) // 随机生成A-Z的大写字母
	}
	return string(key)
}

// 随机UA池
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Mobile Safari/537.36",
}

func getRandomUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	return userAgents[rand.Intn(len(userAgents))]
}

// 伪装Cookies生成
func generateFakeCookies(domain string) []*http.Cookie {
	return []*http.Cookie{
		{
			Name:     "session_id",
			Value:    fmt.Sprintf("fake-session-%d", time.Now().UnixNano()),
			Domain:   domain,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
		},
		{
			Name:     "user_pref",
			Value:    "lang=zh-CN",
			Domain:   domain,
			Path:     "/",
			HttpOnly: false,
		},
	}
}

// 抓取任务处理
func processTask(task *Task) {
	c := colly.NewCollector()

	// 随机UA和伪装Cookies
	c.OnRequest(func(r *colly.Request) {
		ua := getRandomUserAgent()
		r.Headers.Set("User-Agent", ua)

		u, err := url.Parse(r.URL.String())
		if err == nil {
			cookies := generateFakeCookies(u.Host)
			c.SetCookies(u.Host, cookies)
		}

		log.Printf("[INFO] Requesting URL: %s with User-Agent: %s", r.URL.String(), ua)
	})

	// 页面解析1.0
	/*var foundGood, foundBad bool
	c.OnHTML("body", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, "你的网站名称") {
			foundGood = true
		}
		if strings.Contains(e.Text, "你的网站简介") {
			foundBad = true
		}
	})*/

	// 页面解析2.0
	c.OnHTML("body", func(e *colly.HTMLElement) {
		text := strings.ToLower(e.Text)
		if strings.Contains(text, strings.ToLower(keyword1)) && strings.Contains(text, strings.ToLower(keyword2)) {
			task.Result = true
			task.Status = "success"
		} else {
			task.Result = false
			task.Status = "success"
		}
	})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("[ERROR] Failed to request URL: %s, Status Code: %d, Error: %s", r.Request.URL, r.StatusCode, err)
	})

	// 抓取页面
	err := c.Visit(task.URL)
	if err != nil {
		log.Printf("[ERROR] Failed to visit URL: %s, Error: %s", task.URL, err)
		task.Result = false
		task.Status = "failure"
		updateTask(task)
		return
	}

	updateTask(task)

	// 结果判断
	/*task.Result = foundGood && foundBad
	task.Status = "success"
	if !task.Result {
		task.Status = "failure"
	}
	updateTask(task)*/
}

// 更新任务状态
func updateTask(task *Task) {
	taskMu.Lock()
	defer taskMu.Unlock()
	db.Save(task)
}

// API处理：提交任务
func handlePostTask(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-KEY") != apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	task := Task{ID: taskID, URL: req.URL, Status: "processing", CreatedAt: time.Now()}
	db.Create(&task)

	go func() {
		timeout := time.After(5 * time.Minute)
		done := make(chan struct{})
		go func() {
			processTask(&task)
			close(done)
		}()
		select {
		case <-done:
			return
		case <-timeout:
			taskMu.Lock()
			task.Status = "failure"
			task.Result = false
			db.Save(&task)
			taskMu.Unlock()
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
}

// API处理：查询任务
func handleGetTask(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-KEY") != apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id", http.StatusBadRequest)
		return
	}

	var task Task
	if err := db.First(&task, "id = ?", taskID).Error; err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id": task.ID,
		"status":  task.Status,
		"result":  task.Result,
	})
}

func main() {
	var k1, k2, port string
	flag.StringVar(&k1, "n", "example", "网站名称")
	flag.StringVar(&k2, "d", "domain", "网站简介")
	flag.StringVar(&port, "p", "8080", "程序运行端口")
	flag.Parse()

	log.Printf("检测网站名称: %s\n", k1)
	log.Printf("检测网站简介: %s\n", k2)
	log.Printf("运行端口: %s\n", port)

	keyword1 = k1
	keyword2 = k2
	// 初始化
	initDB()
	apiKey = generateAPIKey()
	log.Printf("Server started with API Key: %s", apiKey)

	http.HandleFunc("/api/task", handlePostTask)  // POST任务
	http.HandleFunc("/api/result", handleGetTask) // GET结果

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
