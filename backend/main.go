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
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// --- Models ---

type Task struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"collate:NOCASE;uniqueIndex:idx_task_unique" json:"title"`
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

func initDB() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "calenai.db"
	}
	var err error
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	if err := db.AutoMigrate(&Task{}, &Session{}, &ChatMessage{}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables.")
	}
	fmt.Println("API KEY:", os.Getenv("CEREBRAS_API_KEY"))
	initDB()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173", "http://127.0.0.1:5173", "http://127.0.0.1:4173", "https://calen-ai-cloud.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

	fmt.Println("Server running on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// --- Handlers: Tasks ---

func getTasks(c *gin.Context) {
	var tasks []Task
	db.Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

func createTask(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&task)
	c.JSON(http.StatusCreated, task)
}

func updateTask(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := db.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	Role       string             `json:"role"`
	Content    string             `json:"content,omitempty"`
	ToolCalls  []CerebrasToolCall `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

// CEREBRAS CLOUD AI STRUCT
type CerebrasResponse struct {
	Choices []struct {
		Message CerebrasMessage `json:"message"`
	} `json:"choices"`
}

type CerebrasMessage struct {
	Role      string             `json:"role"`
	Content   *string            `json:"content"`
	ToolCalls []CerebrasToolCall `json:"tool_calls,omitempty"`
}

type CerebrasToolCall struct {
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
	systemPrompt := fmt.Sprintf(
		"You are CalenAI, a sophisticated minimalist AI calendar and task management assistant. "+
			"Current Date: %s. Current Time: %s. "+

			// =====================================================
			// DATE & TIME POLICY
			// =====================================================
			"You MUST resolve all relative dates and dates ranges (for example: 'tomorrow', 'next Tuesday', 'this weekend', 'in two weeks', 'end of this month') using the current date. "+
			"Never ask the user for a date that can be calculated automatically. "+
			"Never invent or estimate dates. "+
			"If the user does not explicitly specify a time when creating a task, NEVER call the add_task tool. "+
			"Instead, ask the user what time they would like. "+
			"Never infer or guess a time under any circumstances. "+
			"You may infer relative dates, but never relative times. "+

			// =====================================================
			// TASK CREATION POLICY
			// =====================================================
			"Only use 'add_task' when the user's intention is to create a completely new task. "+
			"Before creating a task, determine whether the user is actually referring to an existing task. "+
			"If the request is to modify an existing task, NEVER create a duplicate. "+
			"For recurring tasks, call 'add_task' once for every occurrence. "+
			"Do not merely describe recurring tasks—actually create every occurrence using separate tool calls. "+

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
			"For example, if you created a task and then counted tasks, mention both the successful creation and the resulting task count. "+
			"Do not only mention the last tool result. "+

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
			"Keep responses concise, friendly, professional, and action-oriented. "+
			"Avoid unnecessary explanations. "+
			"After successful tool execution, clearly state what changed.",

		time.Now().Format("Monday, January 02, 2006"),
		time.Now().Format("15:04"),
	)

	var messages []ChatMessagePayload
	messages = append(messages, ChatMessagePayload{Role: "system", Content: systemPrompt})
	for _, m := range history {
		messages = append(messages, ChatMessagePayload{Role: m.Role, Content: m.Content})
	}

	// 2. Reasoning Loop (Multi-Think)
	response, err := callCerebrasWithTools(messages, 0)
	if err != nil {
		log.Printf("Cerebras error: %v", err)
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

func callCerebrasWithTools(messages []ChatMessagePayload, depth int) (string, error) {
	apiKey := os.Getenv("CEREBRAS_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("CEREBRAS_API_KEY environment variable is not set")
	}
	url := os.Getenv("CEREBRAS_BASE_URL")
	if url == "" {
		url = "https://api.cerebras.ai/v1/chat/completions"
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
	}
	model := os.Getenv("CEREBRAS_MODEL")
	if model == "" {
		model = "gpt-oss-120b"
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

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)

		if err == nil {

			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode == http.StatusInternalServerError ||
				resp.StatusCode == http.StatusBadGateway ||
				resp.StatusCode == http.StatusServiceUnavailable ||
				resp.StatusCode == http.StatusGatewayTimeout {

				log.Printf(
					"Cerebras returned HTTP %d (attempt %d/%d)",
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
			"Cerebras request failed (attempt %d/%d): %v",
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
		return "", fmt.Errorf("failed to read Cerebras response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf(
			"Cerebras API Error (HTTP %d): %s",
			resp.StatusCode,
			string(body),
		)

		return "", fmt.Errorf(
			"cerebras returned HTTP %d: %s",
			resp.StatusCode,
			string(body),
		)
	}
	var cerebrasResp CerebrasResponse

	if err := json.Unmarshal(body, &cerebrasResp); err != nil {
		return "", fmt.Errorf("failed to parse Cerebras response: %w", err)
	}

	if len(cerebrasResp.Choices) == 0 {
		return "", fmt.Errorf("no response returned from Cerebras")
	}

	if len(cerebrasResp.Choices[0].Message.ToolCalls) > 0 {

		content := ""
		if cerebrasResp.Choices[0].Message.Content != nil {
			content = *cerebrasResp.Choices[0].Message.Content
		}

		assistantMessage := ChatMessagePayload{
			Role:    cerebrasResp.Choices[0].Message.Role,
			Content: content,
		}
		// Preserve assistant tool calls before executing them.
		assistantMessage.ToolCalls = cerebrasResp.Choices[0].Message.ToolCalls

		// Assistant message MUST come before tool responses.
		messages = append(messages, assistantMessage)

		// Execute tool calls.
		for _, tc := range cerebrasResp.Choices[0].Message.ToolCalls {

			log.Printf("AI thinking... Executing tool: %s", tc.Function.Name)
			log.Printf("Raw tool arguments: %s", tc.Function.Arguments)
			log.Printf("Reasoning depth: %d", depth)

			res := executeTool(tc)

			log.Printf("Tool result: %s", res)

			messages = append(messages, ChatMessagePayload{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    res,
			})
		}

		return callCerebrasWithTools(messages, depth+1)
	}
	if cerebrasResp.Choices[0].Message.Content == nil {
		return "Done.", nil
	}
	return *cerebrasResp.Choices[0].Message.Content, nil
}

func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			// Force the parsed clock time into the Local timezone
			// This prevents AI-generated 'Z' suffixes from shifting the time by +5:30 etc.
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format")
}
func applyTaskFilters(query *gorm.DB, args map[string]interface{}) *gorm.DB {

	// Filter by title
	if title, ok := args["title_contains"].(string); ok && title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}

	// Filter by completion status
	if completed, ok := args["completed"].(bool); ok {
		query = query.Where("is_completed = ?", completed)
	}

	// Filter by start date
	if startDate, ok := args["start_date"].(string); ok && startDate != "" {
		query = query.Where("date(start_date_time) >= date(?)", startDate)
	}

	// Filter by end date
	if endDate, ok := args["end_date"].(string); ok && endDate != "" {
		query = query.Where("date(start_date_time) <= date(?)", endDate)
	}

	return query
}
func executeTool(tc CerebrasToolCall) string {
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

		tStart, err := parseFlexibleDate(startStr)
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
			tEnd, err := parseFlexibleDate(endStr)
			if err == nil {
				task.EndDateTime = &tEnd
			}
		}

		// Otherwise default to 1 hour after the start time.
		if task.EndDateTime == nil {
			defaultEnd := tStart.Add(time.Hour)
			task.EndDateTime = &defaultEnd
		}

		if err := db.Create(&task).Error; err != nil {
			return fmt.Sprintf("Failed to create task: %v", err)
		}
		return fmt.Sprintf("Task '%s' added successfully for %s.", title, tStart.Format("Jan 02, 15:04"))
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
		query = query.Where("is_completed = ?", false) // Only count incomplete tasks
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
			t, err := parseFlexibleDate(startStr)
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
	default:
		return "Error: Unknown tool."
	}
}

func getAIContext(c *gin.Context) {
	var tasks []Task
	db.Find(&tasks)
	c.JSON(http.StatusOK, gin.H{
		"current_time": time.Now(),
		"day":          time.Now().Format("Monday"),
		"task_count":   len(tasks),
	})
}
