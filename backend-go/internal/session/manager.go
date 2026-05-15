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

	// Redis 存储（nil = 纯内存模式）
	redisStore *RedisStore
}

// NewSessionManager 创建会话管理器
func NewSessionManager(maxAge time.Duration, maxMessages int, maxTokens int) *SessionManager {
	return NewSessionManagerWithRedis(maxAge, maxMessages, maxTokens, nil)
}

// NewSessionManagerWithRedis 创建带 Redis 支持的会话管理器
func NewSessionManagerWithRedis(maxAge time.Duration, maxMessages int, maxTokens int, redisStore *RedisStore) *SessionManager {
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		responseMapping: make(map[string]string),
		maxAge:          maxAge,
		maxMessages:     maxMessages,
		maxTokens:       maxTokens,
		redisStore:      redisStore,
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

		// 内存未命中 → 尝试从 Redis lazy load
		if sm.redisStore != nil {
			sess, err := sm.loadSessionFromRedis(previousResponseID)
			if err != nil {
				log.Printf("[Session-Redis] 从 Redis 加载会话失败: %v", err)
			} else if sess != nil {
				// 加载成功，写入内存缓存
				sm.sessions[sess.ID] = sess
				sess.LastAccessAt = time.Now()
				log.Printf("[Session-Redis] 从 Redis 加载会话成功: %s", sess.ID)
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

	// 异步写入 Redis
	if sm.redisStore != nil {
		sm.redisStore.SaveSessionAsync(session)
	}

	return session, nil
}

// RecordResponseMapping 记录 responseID 到 sessionID 的映射
func (sm *SessionManager) RecordResponseMapping(responseID, sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.responseMapping[responseID] = sessionID
	log.Printf("[Session-Mapping] 记录映射: %s -> %s", responseID, sessionID)

	if sm.redisStore != nil {
		sm.redisStore.RecordResponseMappingAsync(responseID, sessionID)
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

	if sm.redisStore != nil {
		sm.redisStore.SaveSessionAsync(session)
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

	if sm.redisStore != nil {
		sm.redisStore.SaveSessionAsync(session)
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

	// Redis 模式：清理超限的会话（Redis TTL 已处理时间过期）
	if sm.redisStore != nil {
		sm.cleanupRedisSessions()
		sm.cleanupRedisOrphanedMappings()
	}
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

// loadSessionFromRedis 通过 responseID 从 Redis 加载会话
func (sm *SessionManager) loadSessionFromRedis(previousResponseID string) (*Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 通过 responseID 查找 sessionID
	sessionID, err := sm.redisStore.LookupSessionByResponseID(ctx, previousResponseID)
	if err != nil {
		return nil, fmt.Errorf("lookup response mapping: %w", err)
	}
	if sessionID == "" {
		return nil, nil
	}

	// 加载会话数据
	sess, err := sm.redisStore.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}
	return sess, nil
}

// cleanupRedisSessions 清理 Redis 中超限的会话
func (sm *SessionManager) cleanupRedisSessions() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ids, err := sm.redisStore.GetAllSessionIDs(ctx)
	if err != nil {
		log.Printf("[Session-Cleanup] 获取 Redis 会话列表失败: %v", err)
		return
	}

	removed := 0
	for _, id := range ids {
		sess, err := sm.redisStore.LoadSession(ctx, id)
		if err != nil || sess == nil {
			sm.redisStore.DeleteSession(ctx, id)
			removed++
			continue
		}
		if len(sess.Messages) > sm.maxMessages || sess.TotalTokens > sm.maxTokens {
			sm.redisStore.DeleteSession(ctx, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[Session-Cleanup] Redis 清理: 删除 %d 个超限会话", removed)
	}
}

// cleanupRedisOrphanedMappings 清理 Redis 中指向不存在会话的映射
func (sm *SessionManager) cleanupRedisOrphanedMappings() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 收集内存中的有效 sessionID
	validIDs := make(map[string]struct{})
	for id := range sm.sessions {
		validIDs[id] = struct{}{}
	}
	// 加上 Redis 中的 sessionID
	ids, err := sm.redisStore.GetAllSessionIDs(ctx)
	if err == nil {
		for _, id := range ids {
			validIDs[id] = struct{}{}
		}
	}

	if err := sm.redisStore.CleanOrphanedMappings(ctx, validIDs); err != nil {
		log.Printf("[Session-Cleanup] 清理 Redis 孤立映射失败: %v", err)
	}
}

// Close 关闭 SessionManager（释放 Redis 连接）
func (sm *SessionManager) Close() {
	if sm.redisStore != nil {
		if err := sm.redisStore.Close(); err != nil {
			log.Printf("[Session-Shutdown] 警告: Redis 连接关闭失败: %v", err)
		}
	}
}
