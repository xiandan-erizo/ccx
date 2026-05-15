package session

import (
	"context"
	"testing"
	"time"

	"github.com/BenedictKing/ccx/internal/types"
	"github.com/alicebob/miniredis/v2"
)

func setupRedisStore(t *testing.T) (*RedisStore, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	store, err := NewRedisStore(mr.Addr(), "", 0, 24*time.Hour)
	if err != nil {
		mr.Close()
		t.Fatalf("failed to create Redis store: %v", err)
	}

	return store, func() {
		store.Close()
		mr.Close()
	}
}

func TestRedisStore_SaveAndLoadSession(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	sess := &Session{
		ID:             "sess_test123",
		Messages:       []types.ResponsesItem{{Type: "message", Role: "user", Content: "hello"}},
		LastResponseID: "resp_abc",
		CreatedAt:      time.Now(),
		LastAccessAt:   time.Now(),
		TotalTokens:    100,
	}

	ctx := context.Background()
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	loaded, err := store.LoadSession(ctx, "sess_test123")
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSession returned nil")
	}
	if loaded.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, loaded.ID)
	}
	if loaded.TotalTokens != sess.TotalTokens {
		t.Errorf("expected TotalTokens %d, got %d", sess.TotalTokens, loaded.TotalTokens)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestRedisStore_LoadSession_NotFound(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()
	sess, err := store.LoadSession(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Fatal("expected nil for nonexistent session")
	}
}

func TestRedisStore_RecordAndLookupResponseMapping(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()
	if err := store.RecordResponseMapping(ctx, "resp_xyz", "sess_abc"); err != nil {
		t.Fatalf("RecordResponseMapping failed: %v", err)
	}

	sessionID, err := store.LookupSessionByResponseID(ctx, "resp_xyz")
	if err != nil {
		t.Fatalf("LookupSessionByResponseID failed: %v", err)
	}
	if sessionID != "sess_abc" {
		t.Errorf("expected sessionID sess_abc, got %s", sessionID)
	}

	// 不存在的映射
	sessionID, err = store.LookupSessionByResponseID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionID != "" {
		t.Errorf("expected empty string, got %s", sessionID)
	}
}

func TestRedisStore_DeleteSession(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()
	sess := &Session{ID: "sess_delete", CreatedAt: time.Now(), LastAccessAt: time.Now()}
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	if err := store.DeleteSession(ctx, "sess_delete"); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	loaded, err := store.LoadSession(ctx, "sess_delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestRedisStore_GetAllSessionIDs(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()
	s1 := &Session{ID: "sess_1", CreatedAt: time.Now(), LastAccessAt: time.Now()}
	s2 := &Session{ID: "sess_2", CreatedAt: time.Now(), LastAccessAt: time.Now()}
	_ = store.SaveSession(ctx, s1)
	_ = store.SaveSession(ctx, s2)

	ids, err := store.GetAllSessionIDs(ctx)
	if err != nil {
		t.Fatalf("GetAllSessionIDs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
}

func TestRedisStore_CleanOrphanedMappings(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()
	_ = store.RecordResponseMapping(ctx, "resp_1", "sess_1")
	_ = store.RecordResponseMapping(ctx, "resp_2", "sess_2")

	// 只有 sess_1 有效
	validIDs := map[string]struct{}{"sess_1": {}}
	if err := store.CleanOrphanedMappings(ctx, validIDs); err != nil {
		t.Fatalf("CleanOrphanedMappings failed: %v", err)
	}

	s1, _ := store.LookupSessionByResponseID(ctx, "resp_1")
	s2, _ := store.LookupSessionByResponseID(ctx, "resp_2")
	if s1 != "sess_1" {
		t.Errorf("expected resp_1 -> sess_1, got %s", s1)
	}
	if s2 != "" {
		t.Errorf("expected resp_2 to be cleaned, got %s", s2)
	}
}

func TestSessionManagerWithRedis_GetOrCreateSession(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 预创建一个会话到 Redis（模拟进程重启后恢复）
	sess := &Session{
		ID:             "sess_restore",
		Messages:       []types.ResponsesItem{{Type: "message", Role: "user", Content: "hi"}},
		LastResponseID: "resp_restore",
		CreatedAt:      time.Now().Add(-1 * time.Hour),
		LastAccessAt:   time.Now().Add(-1 * time.Hour),
		TotalTokens:    50,
	}
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
	if err := store.RecordResponseMapping(ctx, "resp_restore", "sess_restore"); err != nil {
		t.Fatalf("RecordResponseMapping failed: %v", err)
	}

	// 创建一个空的 SessionManager（内存中无数据）
	sm := NewSessionManagerWithRedis(24*time.Hour, 100, 100000, store)

	// 通过 previousResponseID 查找，应该从 Redis 加载回来
	loaded, err := sm.GetOrCreateSession("resp_restore")
	if err != nil {
		t.Fatalf("GetOrCreateSession failed: %v", err)
	}
	if loaded.ID != "sess_restore" {
		t.Errorf("expected session ID sess_restore, got %s", loaded.ID)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}

	// 验证已写入内存缓存
	sm.mu.RLock()
	_, exists := sm.sessions["sess_restore"]
	sm.mu.RUnlock()
	if !exists {
		t.Error("session should be cached in memory after Redis load")
	}
}

func TestSessionManagerWithRedis_WriteThrough(t *testing.T) {
	store, cleanup := setupRedisStore(t)
	defer cleanup()

	sm := NewSessionManagerWithRedis(24*time.Hour, 100, 100000, store)

	// 创建新会话
	sess, err := sm.GetOrCreateSession("")
	if err != nil {
		t.Fatalf("GetOrCreateSession failed: %v", err)
	}

	// 等待异步写入完成
	time.Sleep(100 * time.Millisecond)

	// 验证已写入 Redis
	ctx := context.Background()
	loaded, err := store.LoadSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadSession from Redis failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("session not found in Redis")
	}
	if loaded.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, loaded.ID)
	}
}
