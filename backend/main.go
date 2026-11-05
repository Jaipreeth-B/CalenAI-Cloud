package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Task struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"uniqueIndex:idx_task_unique" json:"title"`
	Desc          string         `json:"desc"`
	IsCompleted   bool           `json:"is_completed"`
	StartDateTime *time.Time     `gorm:"uniqueIndex:idx_task_unique" json:"start_date_time"`
	EndDateTime   *time.Time     `json:"end_date_time"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type Session struct {
	ID        string        `gorm:"primaryKey" json:"id"`
	Title     string        `json:"title"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
}

type ChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID string    `gorm:"index" json:"session_id"`
	Role      string    `json:"role"` // 'user', 'assistant', 'system', 'tool'
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

var db *gorm.DB

func getCurrentTime(c *gin.Context) time.Time {
	//log.Printf("All request headers: %+v", c.Request.Header)

	tz := c.GetHeader("X-Timezone")
	//log.Printf("Timezone header: %q", tz)
	if tz == "" {
		now := time.Now().UTC()
		log.Printf("No timezone header. Using UTC: %s", now.Format(time.RFC3339))
		return now
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		now := time.Now().UTC()
		log.Printf("Invalid timezone %q. Falling back to UTC: %s", tz, now.Format(time.RFC3339))
		return now
	}

	now := time.Now().In(loc)

	// log.Printf(
	// 	"Server UTC: %s | Client TZ: %s | Converted: %s",
	// 	time.Now().UTC().Format(time.RFC3339),
	// 	tz,
	// 	now.Format(time.RFC3339),
	// )

	return now
}
func initDB() {
	dbURL := os.Getenv("DATABASE_URL")
	var err error

	if dbURL != "" {
		log.Println("Connecting to cloud PostgreSQL database...")
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dbURL,
			PreferSimpleProtocol: true,
		}), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to PostgreSQL database: %v", err)
		}
		log.Println("Connected to PostgreSQL successfully.")
	} else {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "calenai.db"
		}
		log.Printf("Connecting to local SQLite database (%s)...", dbPath)
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect to SQLite database: %v", err)
		}
	}

	if err := db.AutoMigrate(&Task{}, &Session{}, &ChatMessage{}); err != nil {
		log.Fatalf("failed to auto-migrate database schema: %v", err)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables.")
	}
	//fmt.Println("API KEY:", os.Getenv("CEREBRAS_API_KEY"))
	initDB()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173", "http://127.0.0.1:5173", "http://127.0.0.1:4173", "https://calen-ai-cloud.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-Timezone"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Favicon handler to avoid 404 logs from browsers
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"message":   "CalenAI Backend API is running",
			"frontend":  "http://localhost:5173",
			"endpoints": []string{"/api/tasks", "/api/sessions", "/api/context"},
		})
	})

	// /api root index route
	r.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"message":   "CalenAI API endpoints",
			"endpoints": []string{"/api/tasks", "/api/sessions", "/api/context"},
		})
	})

	api := r.Group("/api")
	{
		// Tasks
		api.GET("/tasks", getTasks)
		api.POST("/tasks", createTask)
		api.PUT("/tasks/:id", updateTask)
		api.DELETE("/tasks/:id", deleteTask)

		// Sessions (Chat History)
		api.GET("/sessions", getSessions)
		api.POST("/sessions", createSession)
		api.GET("/sessions/:id", getSessionMessages)
		api.DELETE("/sessions/:id", deleteSession)

		// Chat
		api.POST("/chat/:session_id", handleChat)

		// Context Helper
		api.GET("/context", getAIContext)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	fmt.Printf("Backend API:  http://localhost:%s\n", port)
	fmt.Printf("Frontend App: http://localhost:5173\n")
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// --- Handlers: Tasks ---

// TaskRequest is the DTO for create/update endpoints;
// it accepts ISO8601 datetime strings with or without timezone.
type TaskRequest struct {
	Title         string `json:"title"`
	Desc          string `json:"desc"`
	IsCompleted   *bool  `json:"is_completed"`
	StartDateTime string `json:"start_date_time"`
	EndDateTime   string `json:"end_date_time"`
}

func getTasks(c *gin.Context) {
	var tasks []Task
	db.Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

func createTask(c *gin.Context) {
	tz := c.GetHeader("X-Timezone")
	if tz == "" {
		tz = "UTC"
	}
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task := Task{
		Title:       req.Title,
		Desc:        req.Desc,
		IsCompleted: false,
	}
	if req.IsCompleted != nil {
		task.IsCompleted = *req.IsCompleted
	}
	if req.StartDateTime != "" {
		if t, err := parseFlexibleDate(req.StartDateTime, tz); err == nil {
			task.StartDateTime = &t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date_time: " + err.Error()})
			return
		}
	}
	if req.EndDateTime != "" {
		if t, err := parseFlexibleDate(req.EndDateTime, tz); err == nil {
			task.EndDateTime = &t
		}
	}
	db.Create(&task)
	c.JSON(http.StatusCreated, task)
}

func updateTask(c *gin.Context) {
	tz := c.GetHeader("X-Timezone")
	if tz == "" {
		tz = "UTC"
	}
	id := c.Param("id")
	var task Task
	if err := db.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	var req TaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Desc != "" {
		task.Desc = req.Desc
	}
	if req.IsCompleted != nil {
		task.IsCompleted = *req.IsCompleted
	}
	if req.StartDateTime != "" {
		if t, err := parseFlexibleDate(req.StartDateTime, tz); err == nil {
			task.StartDateTime = &t
		}
	}
	if req.EndDateTime != "" {
		if t, err := parseFlexibleDate(req.EndDateTime, tz); err == nil {
			task.EndDateTime = &t
		}
	}
	db.Save(&task)
	c.JSON(http.StatusOK, task)
}

func deleteTask(c *gin.Context) {
	id := c.Param("id")
	db.Delete(&Task{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}

// --- Handlers: Sessions ---

func getSessions(c *gin.Context) {
	var sessions []Session
	db.Order("created_at desc").Find(&sessions)
	c.JSON(http.StatusOK, sessions)
}

func createSession(c *gin.Context) {
	session := Session{
		ID:    uuid.New().String(),
		Title: "New Chat " + time.Now().Format("Jan 02, 15:04"),
	}
	db.Create(&session)
	c.JSON(http.StatusCreated, session)
}

func getSessionMessages(c *gin.Context) {
	sessionID := c.Param("id")
	var messages []ChatMessage
	db.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages)
	c.JSON(http.StatusOK, messages)
}

func deleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	db.Where("session_id = ?", sessionID).Delete(&ChatMessage{})
	db.Delete(&Session{}, "id = ?", sessionID)
	c.JSON(http.StatusOK, gin.H{"message": "Session deleted"})
}

// --- AI Chat Logic (Multi-Think) ---

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatMessagePayload struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []OpenRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

// OPENROUTER CLOUD AI STRUCT
type OpenRouterResponse struct {
	Choices []struct {
		Message OpenRouterMessage `json:"message"`
	} `json:"choices"`
}

type OpenRouterMessage struct {
	Role      string               `json:"role"`
	Content   *string              `json:"content"`
	ToolCalls []OpenRouterToolCall `json:"tool_calls,omitempty"`
}

type OpenRouterToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`

	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func handleChat(c *gin.Context) {
	sessionID := c.Param("session_id")
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Persist User Message
	db.Create(&ChatMessage{SessionID: sessionID, Role: "user", Content: req.Message})

	// 1. Context Preparation
	var history []ChatMessage
	//db.Where("session_id = ?", sessionID).Order("created_at asc").Find(&history)
	db.Where("session_id = ?", sessionID).Order("created_at desc").Limit(20).Find(&history)
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	now := getCurrentTime(c)
	systemPrompt := fmt.Sprintf(
		"You are CalenAI, a friendly, sophisticated AI calendar and task management assistant. "+
			"You can converse naturally, answer general questions, and greet the user — not only manage tasks. "+
			"When the user greets you or asks a non-task question, respond warmly and helpfully without calling any tools. "+

			// =====================================================
			// AUTHORITATIVE DATE & TIME
			// =====================================================
			"The following values are authoritative and were generated by the application backend. "+
			"Current Date: %s. "+
			"Current Time (24-hour): %s. "+
			"Current Time (12-hour): %s. "+
			"You MUST treat these values as the single source of truth. "+
			"Never use your own knowledge, memory, training data, or assumptions for the current date or current time. "+
			"If the user asks for the current date, current time, today's date, today's day, or similar questions, ALWAYS answer using the Current Date and Current Time provided above exactly. "+

			// =====================================================
			// DATE & TIME POLICY
			// =====================================================
			"You MUST resolve all relative dates (for example: 'tomorrow', 'day after tomorrow', 'next Tuesday', 'this Saturday', 'this weekend', 'in two weeks', 'end of this month', 'two weeks from today') by computing the exact calendar date from the Current Date above. "+
			"Never ask the user for a date that can be calculated automatically. "+
			"Never invent or estimate dates. "+

			// =====================================================
			// STRICT TIME REQUIREMENT
			// =====================================================
			"A SPECIFIC TIME is REQUIRED before calling add_task. "+
			"A specific time means an exact clock time, for example: '10 AM', '3:30 PM', '14:00', 'noon' (= 12:00 PM), 'midnight' (= 00:00 / 12:00 AM). "+
			"The following words are NOT specific times and must NEVER be used as a time: 'morning', 'afternoon', 'evening', 'night', 'later', 'soon', 'daytime'. "+
			"If the user uses one of these vague words instead of a clock time, ask them for the exact time. "+
			"If the user does not provide a time at all, ask them what time they would like. "+
			"Never infer or guess a time under any circumstances. "+
			"You may infer relative dates, but NEVER infer times. "+

			// =====================================================
			// TASK CREATION POLICY
			// =====================================================
			"Only use 'add_task' when the user's intention is to create a completely new task. "+
			"Before creating a task, determine whether the user is actually referring to an existing task. "+
			"If the request is to modify an existing task, NEVER create a duplicate. "+

			// =====================================================
			// RECURRING TASK POLICY
			// =====================================================
			"For recurring tasks or adding the same task on multiple days (e.g. 'every day', 'every Monday', 'for the whole month', 'daily at 10 AM for 2 weeks'), ALWAYS use 'add_tasks' to add all occurrences in a single call. "+
			"Do NOT call 'add_task' repeatedly or recursively for each day. "+
			"If the user specifies recurring events with a defined period such as 'for the whole month', compute every single date/time in that range and submit them together in 'add_tasks' in a single go. "+
			"If the user specifies 'every day' or similar without a clear end date or count, ask: 'Until when would you like to repeat this, or how many occurrences should I create?' "+
			"Once the duration, count, or end date is known, generate the task list and invoke 'add_tasks' in a single go. "+

			// =====================================================
			// TOOL SELECTION POLICY
			// =====================================================
			"Use 'find_tasks' whenever the user wants to search, inspect, locate, reference, identify, modify, move, reschedule, postpone, advance, delay, rename, edit, change the title, change the description, change the date, change the time, or mark an existing task as completed or incomplete. "+
			"Use 'count_tasks' whenever the user asks how many tasks satisfy some condition. "+
			"Use 'delete_tasks' only when multiple matching tasks should be deleted. "+
			"Use 'delete_task' only when deleting exactly one uniquely identified task. "+
			"Prefer specialized tools ('find_tasks', 'count_tasks', 'delete_tasks') instead of 'get_tasks' whenever possible. "+
			"Use 'get_tasks' only when the user genuinely requests the complete task list. "+

			// =====================================================
			// UPDATE POLICY
			// =====================================================
			"If the user refers to an existing task without specifying an ID (for example: 'my dentist appointment', 'my GRE task', 'that meeting', 'my cricket match'), ALWAYS call 'find_tasks' first. "+
			"After locating the correct task, ALWAYS use 'update_task' to modify it. "+
			"Never use 'add_task' followed by 'delete_task' when 'update_task' can accomplish the same result. "+
			"Always preserve the existing task ID whenever possible. "+
			"Moving or rescheduling a task is an UPDATE operation, not a CREATE operation. "+

			// =====================================================
			// AMBIGUITY POLICY
			// =====================================================
			"If multiple tasks match the user's description and the correct task cannot be identified with high confidence, ask the user for clarification before updating, moving, completing, or deleting anything. "+
			"Never guess which task the user intended. "+

			// =====================================================
			// MULTI-TOOL REASONING
			// =====================================================
			"When multiple tools are required to satisfy a request, execute every required tool before producing the final answer. "+
			"Never stop after the first successful tool if additional tools are still required. "+
			"In your final response, summarize EVERY successful action that was performed. "+
			"For example, if you created two tasks, confirm both. Do not only mention the last tool result. "+

			// =====================================================
			// TOOL SAFETY
			// =====================================================
			"Never fabricate task IDs, dates, tool outputs, or task information. "+
			"Only use information returned by the available tools. "+
			"If a required tool fails, clearly explain the failure instead of pretending it succeeded. "+
			"Do not claim a task was created, updated, moved, or deleted unless the corresponding tool completed successfully. "+

			// =====================================================
			// RESPONSE STYLE
			// =====================================================
			"Keep responses concise, friendly, and action-oriented. "+
			"Avoid unnecessary explanations. "+
			"After successful tool execution, clearly state what changed. "+

			// =====================================================
			// FORMATTING RULES
			// =====================================================
			"NEVER use markdown formatting in your responses. "+
			"Do NOT use ** for bold, * for italics, # for headings, or any other markdown syntax. "+
			"Do NOT use emojis of any kind in your responses. "+
			"Write plain text only. Use plain numbered lists (1. 2. 3.) if listing multiple items. "+

			// =====================================================
			// OVERLAP POLICY
			// =====================================================
			"If the backend returns an OVERLAP_WARNING when you try to add a task, inform the user which tasks overlap and ask how they would like to proceed (keep both, cancel the new one, or reschedule).",

		now.Format("Monday, January 02, 2006"),
		now.Format("15:04"),
		now.Format("03:04 PM"),
	)
	var messages []ChatMessagePayload
	messages = append(messages, ChatMessagePayload{Role: "system", Content: systemPrompt})
	for _, m := range history {
		messages = append(messages, ChatMessagePayload{Role: m.Role, Content: m.Content})
	}

	// 2. Reasoning Loop (Multi-Think)
	tz := c.GetHeader("X-Timezone")
	if tz == "" {
		tz = "UTC"
	}
	response, err := callOpenRouterWithTools(messages, 0, tz)
	if err != nil {
		log.Printf("OpenRouter error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI failure"})
		return
	}

	// Persist Assistant Response
	db.Create(&ChatMessage{SessionID: sessionID, Role: "assistant", Content: response})

	c.JSON(http.StatusOK, gin.H{"response": response})
}

const (
	maxToolCalls = 10
	maxRetries   = 3
)

func callOpenRouterWithTools(messages []ChatMessagePayload, depth int, timezone string) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	url := os.Getenv("OPENROUTER_BASE_URL")
	if url == "" {
		url = "https://openrouter.ai/api/v1/chat/completions"
	}
	if depth >= maxToolCalls {
		return "", fmt.Errorf("agent exceeded maximum tool calls (%d)", maxToolCalls)
	}
	tools := []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "add_task",
				"description": "Add a SINGLE task or calendar event. For recurring tasks, call this tool once for each occurrence. If the user specifies an end time or duration, include end_date_time. Otherwise omit it; the backend will automatically default the event duration to 1 hour.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type": "string",
						},
						"desc": map[string]interface{}{
							"type": "string",
						},
						"start_date_time": map[string]interface{}{
							"type":        "string",
							"description": "Required. ISO8601 format (e.g., 2026-06-02T17:00:00).",
						},
						"end_date_time": map[string]interface{}{
							"type":        "string",
							"description": "Optional. ISO8601 format (e.g., 2026-06-02T18:00:00). Provide only if the user explicitly specifies an end time or duration. Otherwise omit this field.",
						},
					},
					"required": []string{"title", "start_date_time"},
				},
			},
		},

		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "add_tasks",
				"description": "Add MULTIPLE tasks in a single call. Use this instead of calling add_task repeatedly when creating recurring or batch events (e.g. daily workouts for 30 days). Each item in the 'tasks' array follows the same schema as add_task.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"tasks": map[string]interface{}{
							"type":        "array",
							"description": "List of tasks to create.",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"title":           map[string]interface{}{"type": "string"},
									"desc":            map[string]interface{}{"type": "string"},
									"start_date_time": map[string]interface{}{"type": "string", "description": "ISO8601, e.g. 2026-11-01T06:00:00"},
									"end_date_time":   map[string]interface{}{"type": "string", "description": "Optional ISO8601 end time."},
								},
								"required": []string{"title", "start_date_time"},
							},
						},
					},
					"required": []string{"tasks"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_tasks",
				"description": "Retrieve all tasks and events to see current schedule and task IDs.",
				"parameters":  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "find_tasks",
				"description": "Find tasks matching the provided filters.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title_contains": map[string]interface{}{
							"type":        "string",
							"description": "Find tasks whose title contains this text.",
						},
						"start_date": map[string]interface{}{
							"type":        "string",
							"description": "Start of date range (YYYY-MM-DD).",
						},
						"end_date": map[string]interface{}{
							"type":        "string",
							"description": "End of date range (YYYY-MM-DD).",
						},
						"completed": map[string]interface{}{
							"type": "boolean",
						},
					},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "count_tasks",
				"description": "Count tasks matching the provided filters.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title_contains": map[string]interface{}{
							"type": "string",
						},
						"start_date": map[string]interface{}{
							"type": "string",
						},
						"end_date": map[string]interface{}{
							"type": "string",
						},
						"completed": map[string]interface{}{
							"type": "boolean",
						},
					},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "delete_tasks",
				"description": "Delete all tasks matching the provided filters.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title_contains": map[string]interface{}{
							"type": "string",
						},
						"start_date": map[string]interface{}{
							"type": "string",
						},
						"end_date": map[string]interface{}{
							"type": "string",
						},
						"completed": map[string]interface{}{
							"type": "boolean",
						},
					},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "update_task",
				"description": "Update an existing task's properties.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":              map[string]interface{}{"type": "integer"},
						"title":           map[string]interface{}{"type": "string"},
						"desc":            map[string]interface{}{"type": "string"},
						"is_completed":    map[string]interface{}{"type": "boolean"},
						"start_date_time": map[string]interface{}{"type": "string"},
					},
					"required": []string{"id"},
				},
			},
		},

		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "delete_task",
				"description": "Delete a task by its ID.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{"type": "integer"},
					},
					"required": []string{"id"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_current_time",
				"description": "Returns the user's current local date and time.",
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "meta-llama/llama-3.1-8b-instruct"
	}
	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"tools":    tools,
		"stream":   false,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var resp *http.Response
	var req *http.Request
	for attempt := 1; attempt <= maxRetries; attempt++ {

		req, err = http.NewRequest(
			"POST",
			url,
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			return "", err
		}

		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		// OpenRouter recommended headers
		req.Header.Set("HTTP-Referer", "http://localhost:5173")
		req.Header.Set("X-Title", "CalenAI")

		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)

		if err == nil {

			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode == http.StatusInternalServerError ||
				resp.StatusCode == http.StatusBadGateway ||
				resp.StatusCode == http.StatusServiceUnavailable ||
				resp.StatusCode == http.StatusGatewayTimeout {

				log.Printf(
					"OpenRouter returned HTTP %d (attempt %d/%d)",
					resp.StatusCode,
					attempt,
					maxRetries,
				)

				resp.Body.Close()

				if attempt < maxRetries {

					// Respect Retry-After header if provided.
					if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
						if seconds, err := strconv.Atoi(retryAfter); err == nil {
							resp.Body.Close()
							time.Sleep(time.Duration(seconds) * time.Second)
							continue
						}
					}

					resp.Body.Close()

					// Fallback to exponential backoff.
					time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
					continue
				}
			}

			break
		}

		log.Printf(
			"OpenRouter request failed (attempt %d/%d): %v",
			attempt,
			maxRetries,
			err,
		)

		if attempt < maxRetries {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}
	}

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenRouter response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf(
			"OpenRouter API Error (HTTP %d): %s",
			resp.StatusCode,
			string(body),
		)

		return "", fmt.Errorf(
			"openrouter returned HTTP %d: %s",
			resp.StatusCode,
			string(body),
		)
	}
	var openRouterResp OpenRouterResponse

	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenRouter response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no response returned from OpenRouter")
	}

	if len(openRouterResp.Choices[0].Message.ToolCalls) > 0 {

		content := ""
		if openRouterResp.Choices[0].Message.Content != nil {
			content = *openRouterResp.Choices[0].Message.Content
		}

		assistantMessage := ChatMessagePayload{
			Role:    openRouterResp.Choices[0].Message.Role,
			Content: content,
		}
		// Preserve assistant tool calls before executing them.
		assistantMessage.ToolCalls = openRouterResp.Choices[0].Message.ToolCalls

		// Assistant message MUST come before tool responses.
		messages = append(messages, assistantMessage)

		// Execute tool calls.
		for _, tc := range openRouterResp.Choices[0].Message.ToolCalls {

			log.Printf("AI thinking... Executing tool: %s", tc.Function.Name)
			log.Printf("Raw tool arguments: %s", tc.Function.Arguments)
			log.Printf("Reasoning depth: %d", depth)

			res := executeTool(tc, timezone)

			log.Printf("Tool result: %s", res)

			messages = append(messages, ChatMessagePayload{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    res,
			})
		}

		return callOpenRouterWithTools(messages, depth+1, timezone)
	}
	if openRouterResp.Choices[0].Message.Content == nil {
		return "Done.", nil
	}
	return *openRouterResp.Choices[0].Message.Content, nil
}

func parseFlexibleDate(s string, timezone string) (time.Time, error) {
	// Resolve user timezone, fall back to UTC if invalid or empty.
	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}

	// Formats that already contain timezone information.
	timezoneFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
	}

	for _, f := range timezoneFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	// Formats without timezone information — treat as user's local timezone.
	localFormats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, f := range localFormats {
		if t, err := time.ParseInLocation(f, s, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format")
}
func applyTaskFilters(query *gorm.DB, args map[string]interface{}) *gorm.DB {

	// Filter by title (case-insensitive for both Postgres and SQLite)
	if title, ok := args["title_contains"].(string); ok && title != "" {
		if db.Dialector.Name() == "postgres" {
			query = query.Where("title ILIKE ?", "%"+title+"%")
		} else {
			query = query.Where("title LIKE ?", "%"+title+"%")
		}
	}

	// Filter by completion status
	if completed, ok := args["completed"].(bool); ok {
		query = query.Where("is_completed = ?", completed)
	}

	// Filter by start date
	if startDate, ok := args["start_date"].(string); ok && startDate != "" {
		if db.Dialector.Name() == "postgres" {
			query = query.Where("start_date_time::date >= ?::date", startDate)
		} else {
			query = query.Where("date(start_date_time) >= date(?)", startDate)
		}
	}

	// Filter by end date
	if endDate, ok := args["end_date"].(string); ok && endDate != "" {
		if db.Dialector.Name() == "postgres" {
			query = query.Where("start_date_time::date <= ?::date", endDate)
		} else {
			query = query.Where("date(start_date_time) <= date(?)", endDate)
		}
	}

	return query
}
func executeTool(tc OpenRouterToolCall, timezone string) string {
	var args map[string]interface{}

	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf(
			"Error: Failed to parse arguments for tool '%s': %v",
			tc.Function.Name,
			err,
		)
	}
	switch tc.Function.Name {
	case "add_task":
		title, _ := args["title"].(string)
		desc, _ := args["desc"].(string)
		startStr, _ := args["start_date_time"].(string)

		if startStr == "" {
			return "Error: start_date_time is required."
		}

		tStart, err := parseFlexibleDate(startStr, timezone)
		if err != nil {
			return "Error: Invalid start date format. Please use ISO8601 (e.g., 2026-06-02T17:00:00)."
		}

		task := Task{
			Title:         title,
			Desc:          desc,
			StartDateTime: &tStart,
		}

		// Use AI-provided end time if available.
		if endStr, ok := args["end_date_time"].(string); ok && endStr != "" {
			tEnd, err := parseFlexibleDate(endStr, timezone)
			if err == nil {
				task.EndDateTime = &tEnd
			}
		}

		// Otherwise default to 1 hour after the start time.
		if task.EndDateTime == nil {
			defaultEnd := tStart.Add(time.Hour)
			task.EndDateTime = &defaultEnd
		}

		// Check for overlapping tasks before inserting.
		var overlapping []Task
		db.Where(
			"deleted_at IS NULL AND start_date_time < ? AND end_date_time > ?",
			*task.EndDateTime,
			tStart,
		).Find(&overlapping)
		if len(overlapping) > 0 {
			names := ""
			for i, t := range overlapping {
				if i > 0 {
					names += ", "
				}
				names += fmt.Sprintf("'%s' (%s)", t.Title, t.StartDateTime.Format("Jan 02, 15:04"))
			}
			return fmt.Sprintf("OVERLAP_WARNING: The new task '%s' (%s - %s) overlaps with: %s. Ask the user how to proceed.",
				title,
				tStart.Format("Jan 02, 15:04"),
				task.EndDateTime.Format("15:04"),
				names,
			)
		}

		if err := db.Create(&task).Error; err != nil {
			return fmt.Sprintf("Failed to create task: %v", err)
		}
		return fmt.Sprintf("Task '%s' added successfully for %s.", title, tStart.Format("Jan 02, 15:04"))
	case "add_tasks", "create_tasks":
		rawItems, ok := args["tasks"]
		if !ok {
			return "Error: 'tasks' array is required."
		}
		items, ok := rawItems.([]interface{})
		if !ok {
			return "Error: 'tasks' must be an array."
		}

		created := 0
		skipped := 0
		var overlapMsgs []string

		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			title, _ := item["title"].(string)
			desc, _ := item["desc"].(string)
			startStr, _ := item["start_date_time"].(string)
			if startStr == "" {
				skipped++
				continue
			}
			tStart, err := parseFlexibleDate(startStr, timezone)
			if err != nil {
				skipped++
				continue
			}
			defaultEnd := tStart.Add(time.Hour)
			tEnd := &defaultEnd
			if endStr, ok := item["end_date_time"].(string); ok && endStr != "" {
				if t, err := parseFlexibleDate(endStr, timezone); err == nil {
					tEnd = &t
				}
			}

			// Overlap check
			var overlapping []Task
			db.Where("deleted_at IS NULL AND start_date_time < ? AND end_date_time > ?", *tEnd, tStart).Find(&overlapping)
			if len(overlapping) > 0 {
				names := ""
				for i, t := range overlapping {
					if i > 0 {
						names += ", "
					}
					names += fmt.Sprintf("'%s' (%s)", t.Title, t.StartDateTime.Format("Jan 02, 15:04"))
				}
				overlapMsgs = append(overlapMsgs, fmt.Sprintf("'%s' at %s overlaps with %s", title, tStart.Format("Jan 02, 15:04"), names))
				skipped++
				continue
			}

			task := Task{Title: title, Desc: desc, StartDateTime: &tStart, EndDateTime: tEnd}
			if err := db.Create(&task).Error; err == nil {
				created++
			} else {
				skipped++
			}
		}

		result := fmt.Sprintf("%d tasks created successfully.", created)
		if skipped > 0 {
			result += fmt.Sprintf(" %d skipped.", skipped)
		}
		if len(overlapMsgs) > 0 {
			result += " OVERLAP_WARNING: " + fmt.Sprintf("%v", overlapMsgs)
		}
		return result

	case "find_tasks":
		query := applyTaskFilters(db.Model(&Task{}), args)

		var tasks []Task
		query.Find(&tasks)

		data, err := json.Marshal(tasks)
		if err != nil {
			return "Error: Failed to serialize tasks."
		}

		return string(data)
	case "get_tasks":
		var tasks []Task
		db.Find(&tasks)
		data, _ := json.Marshal(tasks)
		return string(data)
	case "count_tasks":
		query := applyTaskFilters(db.Model(&Task{}), args)
		// Note: completion filter is handled by applyTaskFilters via the "completed" arg.
		// Do NOT add a hardcoded is_completed filter here.
		var count int64
		query.Count(&count)

		return fmt.Sprintf("%d", count)
	case "update_task":
		idFloat, _ := args["id"].(float64)
		id := uint(idFloat)
		var task Task
		if err := db.First(&task, id).Error; err != nil {
			return "Error: Task not found."
		}

		if title, ok := args["title"].(string); ok {
			task.Title = title
		}
		if desc, ok := args["desc"].(string); ok {
			task.Desc = desc
		}
		if completed, ok := args["is_completed"].(bool); ok {
			task.IsCompleted = completed
		}
		if startStr, ok := args["start_date_time"].(string); ok {
			t, err := parseFlexibleDate(startStr, timezone)
			if err == nil {

				// Preserve the original duration if an end time already exists.
				if task.StartDateTime != nil && task.EndDateTime != nil {
					duration := task.EndDateTime.Sub(*task.StartDateTime)

					task.StartDateTime = &t

					newEnd := t.Add(duration)
					task.EndDateTime = &newEnd
				} else {
					task.StartDateTime = &t
				}
			}
		}

		db.Save(&task)
		return "Task updated successfully."
	case "delete_tasks":
		query := applyTaskFilters(db.Model(&Task{}), args)

		result := query.Delete(&Task{})

		if result.Error != nil {
			return fmt.Sprintf("Error deleting tasks: %v", result.Error)
		}

		return fmt.Sprintf("Successfully deleted %d task(s).", result.RowsAffected)
	case "delete_task":
		idFloat, _ := args["id"].(float64)
		id := uint(idFloat)
		var task Task
		if err := db.First(&task, id).Error; err != nil {
			log.Printf("AI tried to delete non-existent task ID: %d", id)
			return fmt.Sprintf("Error: Task with ID %d not found.", id)
		}
		db.Delete(&task)
		log.Printf("AI successfully deleted task: %s (ID: %d)", task.Title, id)
		return fmt.Sprintf("Successfully deleted task: %s", task.Title)
	case "get_current_time":
		loc := time.UTC
		if timezone != "" {
			if l, err := time.LoadLocation(timezone); err == nil {
				loc = l
			}
		}
		now := time.Now().In(loc)
		return fmt.Sprintf(
			"Current date and time: %s | Date: %s | Time (24h): %s | Time (12h): %s | Day: %s",
			now.Format(time.RFC3339),
			now.Format("Monday, January 02, 2006"),
			now.Format("15:04"),
			now.Format("03:04 PM"),
			now.Format("Monday"),
		)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'. Available tools: add_task, add_tasks, get_tasks, find_tasks, count_tasks, update_task, delete_task, delete_tasks, get_current_time.", tc.Function.Name)
	}
}

func getAIContext(c *gin.Context) {
	var tasks []Task
	db.Find(&tasks)

	now := getCurrentTime(c)

	c.JSON(http.StatusOK, gin.H{
		"current_time": now,
		"day":          now.Format("Monday"),
		"task_count":   len(tasks),
	})
}
