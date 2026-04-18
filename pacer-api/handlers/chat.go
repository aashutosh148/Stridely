package handlers

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/services"
)

type ChatHandler struct {
	db       *db.Postgres
	agentSvc *services.AgentService
}

func NewChatHandler(database *db.Postgres, agentSvc *services.AgentService) *ChatHandler {
	return &ChatHandler{
		db:       database,
		agentSvc: agentSvc,
	}
}

// Chat handles POST /chat with Server-Sent Events streaming
func (h *ChatHandler) Chat(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	// Validate input
	if req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message is required"})
	}

	// Generate session ID if not provided
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create stream channel
	streamCh := make(chan string, 32)

	// Run agent in goroutine
	go func() {
		defer close(streamCh)
		
		ctx := context.Background()
		response, err := h.agentSvc.RunLoop(ctx, userID, req.Message, streamCh)
		if err != nil {
			streamCh <- fmt.Sprintf(`{"type":"error","msg":"%s"}`, err.Error())
			return
		}

		// Save to database
		if err := h.saveChatMessages(ctx, userID, req.SessionID, req.Message, response); err != nil {
			streamCh <- fmt.Sprintf(`{"type":"error","msg":"failed to save chat: %s"}`, err.Error())
			return
		}

		streamCh <- fmt.Sprintf(`{"type":"done","session_id":"%s"}`, req.SessionID)
	}()

	// Pipe channel to SSE stream
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for chunk := range streamCh {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			w.Flush()
		}
	})

	return nil
}

// GetHistory handles GET /chat/history
func (h *ChatHandler) GetHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	sessionID := c.Query("session_id")

	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	var messages []ChatMessage
	var query string
	var args []interface{}

	if sessionID != "" {
		// Get messages for specific session
		sid, err := uuid.Parse(sessionID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid session ID"})
		}
		query = `SELECT id, user_id, session_id, role, content, tool_calls, created_at
		         FROM chat_messages
		         WHERE user_id = $1 AND session_id = $2
		         ORDER BY created_at ASC`
		args = []interface{}{uid, sid}
	} else {
		// Get recent messages (last 50)
		query = `SELECT id, user_id, session_id, role, content, tool_calls, created_at
		         FROM chat_messages
		         WHERE user_id = $1
		         ORDER BY created_at DESC
		         LIMIT 50`
		args = []interface{}{uid}
	}

	rows, err := h.db.Pool.Query(c.Context(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}
	defer rows.Close()

	for rows.Next() {
		var msg ChatMessage
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.SessionID, &msg.Role, 
		                 &msg.Content, &msg.ToolCalls, &msg.CreatedAt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scan error"})
		}
		messages = append(messages, msg)
	}

	// Reverse if we fetched recent messages
	if sessionID == "" {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return c.JSON(fiber.Map{
		"messages":   messages,
		"session_id": sessionID,
		"count":      len(messages),
	})
}

// saveChatMessages stores user and assistant messages to database
func (h *ChatHandler) saveChatMessages(ctx context.Context, userID, sessionID, userMsg, assistantMsg string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}

	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert user message
	_, err = tx.Exec(ctx, `
		INSERT INTO chat_messages (user_id, session_id, role, content, created_at)
		VALUES ($1, $2, 'user', $3, $4)
	`, uid, sid, userMsg, time.Now())
	if err != nil {
		return err
	}

	// Insert assistant message
	_, err = tx.Exec(ctx, `
		INSERT INTO chat_messages (user_id, session_id, role, content, created_at)
		VALUES ($1, $2, 'assistant', $3, $4)
	`, uid, sid, assistantMsg, time.Now())
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	SessionID uuid.UUID      `json:"session_id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls sql.NullString `json:"tool_calls,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
