package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/BenedictKing/ccx/internal/types"
)

// Session 会话数据结构
type Session struct {
	ID             string                `json:"id"`                // sess_xxxxx
	Messages       []types.ResponsesItem `json:"messages"`         // 完整对话历史
	LastResponseID string                `json:"last_response_id"` // 最后一个 response ID
	CreatedAt      time.Time             `json:"created_at"`
	LastAccessAt   time.Time             `json:"last_access_at"`
	TotalTokens    int                   `json:"total_tokens"`
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions        map[string]*Session // sessionID → Session
	responseMapping map[string]string   // responseID → sessionID
	mu              sync.RWMutex

	// 清理配置
	maxAge      time.Duration // 24小时
	maxMessages int           // 100条
	maxTokens   int           // 100k

	// PG 持久化存储（nil = 纯内存模式）
	pgStore *PGSessionStore
}

// NewSessionManager 创建会话管理器
func NewSessionManager(maxAge time.Duration, maxMessages int, maxTokens int) *SessionManager {
	return NewSessionManagerWithPG(maxAge, maxMessages, maxTokens, nil)
}

// NewSessionManagerWithPG 创建带 PG 持久化支持的会话管理器
func NewSessionManagerWithPG(maxAge time.Duration, maxMessages int, maxTokens int, pgStore *PGSessionStore) *SessionManager {
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		responseMapping: make(map[string]string),
		maxAge:          maxAge,
		maxMessages:     maxMessages,
		maxTokens:       maxTokens,
		pgStore:         pgStore,
	}

	// 启动定期清理
	go sm.cleanupLoop()

	return sm
}

// GetOrCreateSession 获取或创建会话
func (sm *SessionManager) GetOrCreateSession(previousResponseID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 如果提供了 previousResponseID，尝试查找对应的会话
	if previousResponseID != "" {
		if sessionID, ok := sm.responseMapping[previousResponseID]; ok {
			if session, exists := sm.sessions[sessionID]; exists {
				session.LastAccessAt = time.Now()
				return session, nil
			}
		}

		// 内存未命中 → 尝试从 PG lazy load
		if sm.pgStore != nil {
			sess, err := sm.loadSessionFromPG(previousResponseID)
			if err != nil {
				log.Printf("[Session-PG] 从 PG 加载会话失败: %v", err)
			} else if sess != nil {
				// 加载成功，写入内存缓存
				sm.sessions[sess.ID] = sess
				sess.LastAccessAt = time.Now()
				log.Printf("[Session-PG] 从 PG 加载会话成功: %s", sess.ID)
				return sess, nil
			}
		}

		// 如果找不到对应会话，返回错误
		return nil, fmt.Errorf("无效的 previous_response_id: %s", previousResponseID)
	}

	// 创建新会话
	sessionID := generateID("sess")
	session := &Session{
		ID:           sessionID,
		Messages:     []types.ResponsesItem{},
		CreatedAt:    time.Now(),
		LastAccessAt: time.Now(),
		TotalTokens:  0,
	}

	sm.sessions[sessionID] = session
	log.Printf("[Session-Create] 创建新会话: %s", sessionID)

	// 异步写入 PG
	if sm.pgStore != nil {
		sm.pgStore.SaveSessionAsync(session)
	}

	return session, nil
}

// RecordResponseMapping 记录 responseID 到 sessionID 的映射
func (sm *SessionManager) RecordResponseMapping(responseID, sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.responseMapping[responseID] = sessionID
	log.Printf("[Session-Mapping] 记录映射: %s -> %s", responseID, sessionID)

	if sm.pgStore != nil {
		sm.pgStore.RecordResponseMappingAsync(responseID, sessionID)
	}
}

// AppendMessage 追加消息到会话
func (sm *SessionManager) AppendMessage(sessionID string, item types.ResponsesItem, tokensUsed int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	session.Messages = append(session.Messages, item)
	session.TotalTokens += tokensUsed
	session.LastAccessAt = time.Now()

	if sm.pgStore != nil {
		sm.pgStore.SaveSessionAsync(session)
	}

	return nil
}

// UpdateLastResponseID 更新会话的最后一个 responseID
func (sm *SessionManager) UpdateLastResponseID(sessionID, responseID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	session.LastResponseID = responseID

	if sm.pgStore != nil {
		sm.pgStore.SaveSessionAsync(session)
	}

	return nil
}

// GetSession 获取会话（只读）
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", sessionID)
	}

	return cloneSession(session)
}

func cloneSession(src *Session) (*Session, error) {
	if src == nil {
		return nil, nil
	}

	cloned := *src
	if len(src.Messages) == 0 {
		cloned.Messages = []types.ResponsesItem{}
		return &cloned, nil
	}

	payload, err := json.Marshal(src.Messages)
	if err != nil {
		return nil, fmt.Errorf("clone session messages failed: %w", err)
	}

	var messages []types.ResponsesItem
	if err := json.Unmarshal(payload, &messages); err != nil {
		return nil, fmt.Errorf("clone session messages failed: %w", err)
	}

	cloned.Messages = messages
	return &cloned, nil
}

// cleanupLoop 定期清理过期会话
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanup()
	}
}

// cleanup 执行清理逻辑
func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	removedSessions := 0
	removedMappings := 0

	// 清理过期会话
	for sessionID, session := range sm.sessions {
		shouldRemove := false

		// 时间过期
		if now.Sub(session.LastAccessAt) > sm.maxAge {
			shouldRemove = true
			log.Printf("[Session-Cleanup] 清理过期会话 (时间): %s (最后访问: %v 前)", sessionID, now.Sub(session.LastAccessAt))
		}

		// 消息数超限
		if len(session.Messages) > sm.maxMessages {
			shouldRemove = true
			log.Printf("[Session-Cleanup] 清理过期会话 (消息数): %s (%d 条)", sessionID, len(session.Messages))
		}

		// Token 超限
		if session.TotalTokens > sm.maxTokens {
			shouldRemove = true
			log.Printf("[Session-Cleanup] 清理过期会话 (Token): %s (%d tokens)", sessionID, session.TotalTokens)
		}

		if shouldRemove {
			delete(sm.sessions, sessionID)
			removedSessions++
		}
	}

	// 清理孤立的 responseID 映射
	for responseID, sessionID := range sm.responseMapping {
		if _, exists := sm.sessions[sessionID]; !exists {
			delete(sm.responseMapping, responseID)
			removedMappings++
		}
	}

	if removedSessions > 0 || removedMappings > 0 {
		log.Printf("[Session-Cleanup] 清理完成: 删除 %d 个会话, %d 个映射", removedSessions, removedMappings)
		log.Printf("[Session-Stats] 当前活跃会话: %d 个, 映射: %d 个", len(sm.sessions), len(sm.responseMapping))
	}

	// PG 模式：清理过期会话和孤立映射
	if sm.pgStore != nil {
		sm.cleanupPGSessions()
		sm.cleanupPGOrphanedMappings()
	}
}

// GetSessionByResponseID 通过 responseID 只读查找 session（不创建新 session）
func (sm *SessionManager) GetSessionByResponseID(responseID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionID, ok := sm.responseMapping[responseID]
	if !ok {
		return nil, fmt.Errorf("未找到 responseID 对应的会话: %s", responseID)
	}

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("会话已过期: %s", sessionID)
	}

	return cloneSession(session)
}

// CreateCompactedSession 创建一个压缩后的 session 并记录 responseID 映射
func (sm *SessionManager) CreateCompactedSession(responseID string, messages []types.ResponsesItem, totalTokens int) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := generateID("sess")
	session := &Session{
		ID:             sessionID,
		Messages:       messages,
		LastResponseID: responseID,
		CreatedAt:      time.Now(),
		LastAccessAt:   time.Now(),
		TotalTokens:    totalTokens,
	}

	sm.sessions[sessionID] = session
	sm.responseMapping[responseID] = sessionID
	log.Printf("[Session-Compact] 创建压缩会话: %s, responseID: %s", sessionID, responseID)

	return sessionID
}

// GetStats 获取统计信息
func (sm *SessionManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"total_sessions": len(sm.sessions),
		"total_mappings": len(sm.responseMapping),
	}
}

// generateID 生成唯一ID
func generateID(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// 降级方案：使用时间戳
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))
}

// loadSessionFromPG 通过 responseID 从 PG 加载会话
func (sm *SessionManager) loadSessionFromPG(previousResponseID string) (*Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 通过 responseID 查找 sessionID
	sessionID, err := sm.pgStore.LookupSessionByResponseID(ctx, previousResponseID)
	if err != nil {
		return nil, fmt.Errorf("lookup response mapping: %w", err)
	}
	if sessionID == "" {
		return nil, nil
	}

	// 加载会话数据
	sess, err := sm.pgStore.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	return sess, nil
}

// cleanupPGSessions 清理 PG 中过期的会话
func (sm *SessionManager) cleanupPGSessions() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	removed, err := sm.pgStore.CleanupExpiredSessions(ctx, sm.maxAge, sm.maxMessages, sm.maxTokens)
	if err != nil {
		log.Printf("[Session-Cleanup] PG 清理过期会话失败: %v", err)
	} else if removed > 0 {
		log.Printf("[Session-Cleanup] PG 清理: 删除 %d 个过期会话", removed)
	}
}

// cleanupPGOrphanedMappings 清理 PG 中指向不存在会话的映射
func (sm *SessionManager) cleanupPGOrphanedMappings() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sm.pgStore.CleanOrphanedMappings(ctx); err != nil {
		log.Printf("[Session-Cleanup] 清理 PG 孤立映射失败: %v", err)
	}
}

// Close 关闭 SessionManager（释放 PG 连接）
func (sm *SessionManager) Close() {
	if sm.pgStore != nil {
		sm.pgStore.Close()
	}
}
