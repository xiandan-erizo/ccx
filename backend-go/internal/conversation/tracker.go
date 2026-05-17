package conversation

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

type Conversation struct {
	ID             string    `json:"id"`
	Kind           string    `json:"kind"`
	UserID         string    `json:"userId"`
	RawUserID      string    `json:"rawUserId,omitempty"`
	Title          string    `json:"title,omitempty"`
	GeneratedTitle string    `json:"-"`
	FallbackTitle  string    `json:"-"`
	SessionID      string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	LastActiveAt   time.Time `json:"lastActiveAt"`
	RequestCount   int       `json:"requestCount"`
	Models         []string  `json:"models"`
	CurrentChannel int       `json:"currentChannel"`
	ChannelName    string    `json:"channelName"`
	Status         string    `json:"status"`
	LastModel      string    `json:"lastModel"`
	LastRequestID  string    `json:"lastRequestId"`
}

func (conv *Conversation) recomputeTitle() {
	if conv.GeneratedTitle != "" {
		const maxTitleRunes = 80
		if len([]rune(conv.GeneratedTitle)) < maxTitleRunes && conv.FallbackTitle != "" {
			conv.Title = composeTitleWithFallback(conv.GeneratedTitle, conv.FallbackTitle)
			return
		}
		conv.Title = truncateTitle(conv.GeneratedTitle)
		return
	}
	conv.Title = conv.FallbackTitle
}

type pendingTitle struct {
	title     string
	createdAt time.Time
}

type ConversationTracker struct {
	mu               sync.RWMutex
	conversations    map[string]*Conversation
	sessionMapping   map[string]string        // sessionID → conversationID (for Responses)
	userMapping      map[string]string        // kind:userID → conversationID (for Chat/Messages/Gemini)
	pendingTitles    map[string]*pendingTitle // kind:userID → title (title 请求先于对话创建时暂存)
	idleTTL          time.Duration
	expireTTL        time.Duration
	maxConversations int
	persistPath      string
	dirty            bool
	stopCh           chan struct{}
	stopOnce         sync.Once
}

func NewConversationTracker(idleTTL, expireTTL time.Duration, persistPath ...string) *ConversationTracker {
	path := ""
	if len(persistPath) > 0 {
		path = persistPath[0]
	}

	ct := &ConversationTracker{
		conversations:    make(map[string]*Conversation),
		sessionMapping:   make(map[string]string),
		userMapping:      make(map[string]string),
		pendingTitles:    make(map[string]*pendingTitle),
		idleTTL:          idleTTL,
		expireTTL:        expireTTL,
		maxConversations: 100,
		persistPath:      path,
		stopCh:           make(chan struct{}),
	}

	if path != "" {
		ct.loadFromDisk()
	}

	go ct.cleanupLoop()
	if path != "" {
		go ct.persistLoop()
	}
	return ct
}

func (ct *ConversationTracker) Track(kind, userID, model string, channelIndex int, channelName, sessionID, lastUserMessage string, userMessageCount int) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	convID := ct.resolveConversationID(kind, userID, sessionID)
	now := time.Now()

	conv, exists := ct.conversations[convID]
	if !exists {
		conv = &Conversation{
			ID:        convID,
			Kind:      kind,
			UserID:    maskUserID(userID),
			RawUserID: userID,
			CreatedAt: now,
			Status:    "active",
			Models:    []string{},
		}
		ct.conversations[convID] = conv

		if sessionID != "" {
			conv.SessionID = sessionID
			ct.sessionMapping[sessionID] = convID
		}
		compositeKey := kind + ":" + userID
		ct.userMapping[compositeKey] = convID

		// 应用先于对话创建到达的 pending title
		if pt, ok := ct.pendingTitles[compositeKey]; ok {
			conv.GeneratedTitle = pt.title
			conv.recomputeTitle()
			delete(ct.pendingTitles, compositeKey)
		}
	}

	conv.LastActiveAt = now
	if userMessageCount > 0 {
		conv.RequestCount = userMessageCount
	} else {
		conv.RequestCount++
	}
	conv.CurrentChannel = channelIndex
	conv.ChannelName = channelName
	conv.LastModel = model
	conv.Status = "active"

	if !containsString(conv.Models, model) {
		conv.Models = append(conv.Models, model)
	}

	if fallback := fallbackTitleFromUserMessage(lastUserMessage); fallback != "" {
		conv.FallbackTitle = fallback
		conv.recomputeTitle()
	}
	ct.dirty = true
}

func (ct *ConversationTracker) UpdateTitle(kind, userID, title string) bool {
	if userID == "" || title == "" {
		return false
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	convID := ct.resolveConversationID(kind, userID, "")
	conv, exists := ct.conversations[convID]
	if !exists {
		// 对话尚未创建（title 请求先于正常请求到达），暂存 title
		compositeKey := kind + ":" + userID
		ct.pendingTitles[compositeKey] = &pendingTitle{
			title:     strings.TrimSpace(title),
			createdAt: time.Now(),
		}
		return true
	}

	conv.GeneratedTitle = strings.TrimSpace(title)
	conv.recomputeTitle()
	ct.dirty = true
	return true
}

func (ct *ConversationTracker) UpdateStatus(kind, userID, status string) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return
	}
	conv, ok := ct.conversations[convID]
	if !ok {
		return
	}
	conv.Status = status
	conv.LastActiveAt = time.Now()
	ct.dirty = true
}

func (ct *ConversationTracker) SetLastRequestID(kind, userID, requestID string) {
	if userID == "" {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return
	}
	conv, ok := ct.conversations[convID]
	if !ok {
		return
	}
	conv.LastRequestID = requestID
	ct.dirty = true
}

func (ct *ConversationTracker) GetActiveConversations(kindFilter string) []*Conversation {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make([]*Conversation, 0, len(ct.conversations))
	for _, conv := range ct.conversations {
		if kindFilter != "" && conv.Kind != kindFilter {
			continue
		}
		result = append(result, conv)
	}
	return result
}

func (ct *ConversationTracker) GetConversation(id string) (*Conversation, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	conv, ok := ct.conversations[id]
	return conv, ok
}

func (ct *ConversationTracker) GetConversationByUser(kind, userID string) (*Conversation, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	compositeKey := kind + ":" + userID
	convID, ok := ct.userMapping[compositeKey]
	if !ok {
		return nil, false
	}
	conv, ok := ct.conversations[convID]
	return conv, ok
}

func (ct *ConversationTracker) Stop() {
	ct.stopOnce.Do(func() {
		close(ct.stopCh)
		if ct.persistPath != "" {
			ct.flushToDisk()
		}
	})
}

func (ct *ConversationTracker) resolveConversationID(kind, userID, sessionID string) string {
	if sessionID != "" {
		if convID, ok := ct.sessionMapping[sessionID]; ok {
			return convID
		}
		return sessionID
	}

	compositeKey := kind + ":" + userID
	if convID, ok := ct.userMapping[compositeKey]; ok {
		return convID
	}

	hash := sha256.Sum256([]byte(compositeKey))
	return "conv_" + hex.EncodeToString(hash[:6])
}

func (ct *ConversationTracker) loadFromDisk() {
	items, skipped, err := loadPersistedState(ct.persistPath, ct.expireTTL)
	if err != nil {
		log.Printf("[ConversationTracker-Load] 读取失败: %v", err)
		return
	}
	if len(items) == 0 && skipped == 0 {
		return
	}

	for _, item := range items {
		models := item.Models
		if models == nil {
			models = []string{}
		}

		conv := &Conversation{
			ID:             item.ID,
			Kind:           item.Kind,
			UserID:         item.UserID,
			RawUserID:      item.RawUserID,
			Title:          item.Title,
			GeneratedTitle: item.GeneratedTitle,
			FallbackTitle:  item.FallbackTitle,
			SessionID:      item.SessionID,
			RequestCount:   item.RequestCount,
			Models:         models,
			CurrentChannel: item.CurrentChannel,
			ChannelName:    item.ChannelName,
			LastModel:      item.LastModel,
			LastRequestID:  item.LastRequestID,
			CreatedAt:      item.CreatedAt,
			LastActiveAt:   item.LastActiveAt,
			Status:         "idle",
		}
		ct.conversations[item.ID] = conv

		compositeKey := item.Kind + ":" + item.RawUserID
		ct.userMapping[compositeKey] = item.ID
		if item.SessionID != "" {
			ct.sessionMapping[item.SessionID] = item.ID
		}
	}
	log.Printf("[ConversationTracker-Load] 恢复 %d 个对话, 跳过 %d 个过期条目", len(items), skipped)
}

func (ct *ConversationTracker) persistLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-ticker.C:
			ct.flushToDisk()
		}
	}
}

func (ct *ConversationTracker) flushToDisk() {
	ct.mu.Lock()
	if !ct.dirty {
		ct.mu.Unlock()
		return
	}
	ct.dirty = false
	snapshot := make(map[string]*Conversation, len(ct.conversations))
	for id, conv := range ct.conversations {
		c := *conv
		if len(conv.Models) > 0 {
			c.Models = make([]string, len(conv.Models))
			copy(c.Models, conv.Models)
		}
		snapshot[id] = &c
	}
	ct.mu.Unlock()

	if err := savePersistedState(ct.persistPath, snapshot); err != nil {
		ct.mu.Lock()
		ct.dirty = true
		ct.mu.Unlock()
		log.Printf("[ConversationTracker-Save] 写入失败: %v", err)
	}
}

func (ct *ConversationTracker) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ct.stopCh:
			return
		case <-ticker.C:
			ct.cleanup()
		}
	}
}

func (ct *ConversationTracker) cleanup() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := time.Now()
	var removed int

	for id, conv := range ct.conversations {
		idleDuration := now.Sub(conv.LastActiveAt)

		if idleDuration > ct.expireTTL {
			ct.removeConversation(id, conv)
			removed++
		} else if idleDuration > ct.idleTTL && conv.Status != "idle" {
			conv.Status = "idle"
		}
	}

	// 清理超时的 pending titles（超过 2 分钟未被消费则丢弃）
	const pendingTitleTTL = 2 * time.Minute
	for key, pt := range ct.pendingTitles {
		if now.Sub(pt.createdAt) > pendingTitleTTL {
			delete(ct.pendingTitles, key)
		}
	}

	// 数量上限裁剪：超过 maxConversations 时按 LastActiveAt 从旧到新删除
	if ct.maxConversations > 0 && len(ct.conversations) > ct.maxConversations {
		type entry struct {
			id   string
			conv *Conversation
		}
		entries := make([]entry, 0, len(ct.conversations))
		for id, conv := range ct.conversations {
			entries = append(entries, entry{id, conv})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].conv.LastActiveAt.Before(entries[j].conv.LastActiveAt)
		})
		excess := len(entries) - ct.maxConversations
		for i := 0; i < excess; i++ {
			ct.removeConversation(entries[i].id, entries[i].conv)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[ConversationTracker-Cleanup] 清理 %d 个过期对话, 剩余 %d", removed, len(ct.conversations))
	}
}

func (ct *ConversationTracker) removeConversation(id string, conv *Conversation) {
	delete(ct.conversations, id)

	compositeKey := conv.Kind + ":" + conv.RawUserID
	if ct.userMapping[compositeKey] == id {
		delete(ct.userMapping, compositeKey)
	}

	for sessID, convID := range ct.sessionMapping {
		if convID == id {
			delete(ct.sessionMapping, sessID)
		}
	}
	ct.dirty = true
}

func maskUserID(userID string) string {
	if len(userID) <= 8 {
		return userID[:1] + "***"
	}
	if idx := strings.Index(userID, "_session_"); idx >= 0 {
		sessionPart := userID[idx+9:]
		if len(sessionPart) > 8 {
			sessionPart = sessionPart[:8]
		}
		return "sess:" + sessionPart
	}
	if len(userID) > 20 {
		return userID[:8] + "..." + userID[len(userID)-4:]
	}
	return userID[:4] + "***" + userID[len(userID)-4:]
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func fallbackTitleFromUserMessage(message string) string {
	msg := strings.ReplaceAll(message, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", "")
	msg = strings.TrimSpace(msg)
	return truncateTitle(msg)
}

func composeTitleWithFallback(title, fallback string) string {
	title = strings.TrimSpace(title)
	fallback = strings.TrimSpace(fallback)
	if title == "" || fallback == "" {
		return title
	}
	if title == fallback || strings.Contains(title, " — "+fallback) {
		return truncateTitle(title)
	}
	combined := title + " — " + fallback
	return truncateTitle(combined)
}

func truncateTitle(title string) string {
	const maxTitleRunes = 80
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	runes := []rune(title)
	if len(runes) > maxTitleRunes {
		return string(runes[:maxTitleRunes]) + "..."
	}
	return title
}
