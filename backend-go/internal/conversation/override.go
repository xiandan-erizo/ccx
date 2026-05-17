package conversation

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type ChannelEntry struct {
	ChannelIndex int    `json:"channelIndex"`
	ChannelName  string `json:"channelName"`
}

type ChannelSequenceOverride struct {
	ConversationID string         `json:"conversationId"`
	Kind           string         `json:"kind"`
	UserID         string         `json:"userID"`
	Sequence       []ChannelEntry `json:"sequence"`
	SetAt          time.Time      `json:"setAt"`
	ExpiresAt      time.Time      `json:"expiresAt"`
}

type OverrideManager struct {
	mu        sync.RWMutex
	overrides map[string]*ChannelSequenceOverride // conversationID → override
	userIndex map[string]string                   // kind:userID → conversationID
	ttl       time.Duration
	stopCh    chan struct{}
}

func NewOverrideManager(ttl time.Duration) *OverrideManager {
	om := &OverrideManager{
		overrides: make(map[string]*ChannelSequenceOverride),
		userIndex: make(map[string]string),
		ttl:       ttl,
		stopCh:    make(chan struct{}),
	}
	go om.cleanupLoop()
	return om
}

func (om *OverrideManager) SetOverride(conversationID, kind, userID string, sequence []ChannelEntry) error {
	if len(sequence) == 0 {
		return fmt.Errorf("sequence cannot be empty")
	}
	if conversationID == "" || kind == "" || userID == "" {
		return fmt.Errorf("conversationID, kind, and userID are required")
	}

	om.mu.Lock()
	defer om.mu.Unlock()

	now := time.Now()
	override := &ChannelSequenceOverride{
		ConversationID: conversationID,
		Kind:           kind,
		UserID:         userID,
		Sequence:       sequence,
		SetAt:          now,
		ExpiresAt:      now.Add(om.ttl),
	}

	om.overrides[conversationID] = override
	compositeKey := kind + ":" + userID
	om.userIndex[compositeKey] = conversationID

	log.Printf("[OverrideManager-Set] 设置覆盖: conv=%s, kind=%s, 序列长度=%d, 过期=%s",
		conversationID, kind, len(sequence), override.ExpiresAt.Format("15:04:05"))

	return nil
}

func (om *OverrideManager) GetOverride(conversationID string) (*ChannelSequenceOverride, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	override, ok := om.overrides[conversationID]
	if !ok {
		return nil, false
	}
	if time.Now().After(override.ExpiresAt) {
		return nil, false
	}
	return override, true
}

func (om *OverrideManager) GetOverrideForUser(kind, userID string) ([]ChannelEntry, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	compositeKey := kind + ":" + userID
	convID, ok := om.userIndex[compositeKey]
	if !ok {
		return nil, false
	}

	override, ok := om.overrides[convID]
	if !ok {
		return nil, false
	}
	if time.Now().After(override.ExpiresAt) {
		return nil, false
	}
	return override.Sequence, true
}

func (om *OverrideManager) RemoveOverride(conversationID string) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	override, ok := om.overrides[conversationID]
	if !ok {
		return false
	}

	compositeKey := override.Kind + ":" + override.UserID
	delete(om.userIndex, compositeKey)
	delete(om.overrides, conversationID)

	log.Printf("[OverrideManager-Remove] 移除覆盖: conv=%s", conversationID)
	return true
}

func (om *OverrideManager) GetAllOverrides() map[string]*ChannelSequenceOverride {
	om.mu.RLock()
	defer om.mu.RUnlock()

	now := time.Now()
	result := make(map[string]*ChannelSequenceOverride, len(om.overrides))
	for id, override := range om.overrides {
		if now.Before(override.ExpiresAt) {
			result[id] = override
		}
	}
	return result
}

func (om *OverrideManager) RefreshTTL(conversationID string) bool {
	om.mu.Lock()
	defer om.mu.Unlock()

	override, ok := om.overrides[conversationID]
	if !ok {
		return false
	}
	override.ExpiresAt = time.Now().Add(om.ttl)
	return true
}

func (om *OverrideManager) Stop() {
	close(om.stopCh)
}

func (om *OverrideManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-om.stopCh:
			return
		case <-ticker.C:
			om.cleanup()
		}
	}
}

func (om *OverrideManager) cleanup() {
	om.mu.Lock()
	defer om.mu.Unlock()

	now := time.Now()
	var removed int

	for id, override := range om.overrides {
		if now.After(override.ExpiresAt) {
			compositeKey := override.Kind + ":" + override.UserID
			delete(om.userIndex, compositeKey)
			delete(om.overrides, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[OverrideManager-Cleanup] 清理 %d 个过期覆盖, 剩余 %d", removed, len(om.overrides))
	}
}
