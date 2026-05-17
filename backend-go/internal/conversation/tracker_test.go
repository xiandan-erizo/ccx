package conversation

import (
	"strings"
	"testing"
	"time"
)

func TestConversationTracker_Track(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user123", "claude-sonnet-4-20250514", 0, "primary", "", "", 0)

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}

	conv := convs[0]
	if conv.Kind != "chat" {
		t.Errorf("expected kind=chat, got %s", conv.Kind)
	}
	if conv.RequestCount != 1 {
		t.Errorf("expected requestCount=1, got %d", conv.RequestCount)
	}
	if conv.ChannelName != "primary" {
		t.Errorf("expected channelName=primary, got %s", conv.ChannelName)
	}
	if conv.LastModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected lastModel=claude-sonnet-4-20250514, got %s", conv.LastModel)
	}
}

func TestConversationTracker_UpdateTitle(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("messages", "session-123", "claude-opus-4-7", 0, "primary", "", "", 0)
	if !ct.UpdateTitle("messages", "session-123", "Build docs preview") {
		t.Fatal("expected UpdateTitle to update existing conversation")
	}

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].Title != "Build docs preview" {
		t.Errorf("expected title=Build docs preview, got %s", convs[0].Title)
	}
	if convs[0].RequestCount != 1 {
		t.Errorf("expected requestCount=1, got %d", convs[0].RequestCount)
	}
}

func TestConversationTracker_UpdateTitleMissingConversation(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	// title 请求先于对话创建到达时，应缓存 title 并返回 true
	if !ct.UpdateTitle("messages", "session-123", "Build docs preview") {
		t.Fatal("expected UpdateTitle to accept pending title")
	}
	if convs := ct.GetActiveConversations(""); len(convs) != 0 {
		t.Fatalf("expected no conversation to be created by UpdateTitle, got %d", len(convs))
	}

	// 后续 Track 创建对话时应自动应用 pending title
	ct.Track("messages", "session-123", "claude-opus-4-7", 0, "primary", "", "", 0)
	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation after Track, got %d", len(convs))
	}
	if !strings.Contains(convs[0].Title, "Build docs preview") {
		t.Fatalf("expected pending title to be applied, got %q", convs[0].Title)
	}
}

func TestConversationTracker_PersistAndRestore(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/state.json"

	ct := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	ct.Track("messages", "user-abc", "claude-opus-4-7", 0, "primary", "", "你好世界", 1)
	ct.UpdateTitle("messages", "user-abc", "确认驾驶舱对话卡片保存时长")
	ct.Stop()

	ct2 := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	defer ct2.Stop()

	convs := ct2.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation after restore, got %d", len(convs))
	}
	if convs[0].Title != "确认驾驶舱对话卡片保存时长 — 你好世界" {
		t.Errorf("expected restored title, got %q", convs[0].Title)
	}
	if convs[0].CreatedAt.IsZero() {
		t.Error("expected non-zero createdAt after restore")
	}

	ct2.Track("messages", "user-abc", "claude-opus-4-7", 1, "backup", "", "", 2)
	convs2 := ct2.GetActiveConversations("")
	if len(convs2) != 1 {
		t.Fatalf("expected mapping to work after restore, got %d conversations", len(convs2))
	}
	if convs2[0].CurrentChannel != 1 {
		t.Errorf("expected channel=1 after re-track, got %d", convs2[0].CurrentChannel)
	}
}

func TestConversationTracker_PersistAndRestoreSessionMapping(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/state.json"

	ct := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	ct.Track("responses", "user-abc", "model-a", 0, "primary", "sess-1", "第一轮", 1)
	ct.Stop()

	ct2 := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	defer ct2.Stop()
	ct2.Track("responses", "user-abc", "model-b", 1, "backup", "sess-1", "第二轮", 2)

	convs := ct2.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected restored session mapping to keep one conversation, got %d", len(convs))
	}
	if convs[0].CurrentChannel != 1 {
		t.Errorf("expected updated channel after restored session mapping, got %d", convs[0].CurrentChannel)
	}
}

func TestConversationTracker_StatusUpdatePersists(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/state.json"

	ct := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	ct.Track("messages", "user-abc", "model-a", 0, "primary", "", "第一轮", 1)
	ct.Stop()

	ct2 := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	ct2.UpdateStatus("messages", "user-abc", "idle")
	ct2.Stop()

	ct3 := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	defer ct3.Stop()
	convs := ct3.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation after status restore, got %d", len(convs))
	}
	if convs[0].Status != "idle" {
		t.Errorf("expected persisted status idle, got %s", convs[0].Status)
	}
}

func TestConversationTracker_TTLFilterOnLoad(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/state.json"

	ct := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	ct.Track("messages", "user-fresh", "model-a", 0, "ch1", "", "新对话", 1)
	ct.Track("messages", "user-old", "model-b", 0, "ch2", "", "旧对话", 1)

	ct.mu.Lock()
	for _, conv := range ct.conversations {
		if conv.RawUserID == "user-old" {
			conv.LastActiveAt = time.Now().Add(-3 * time.Hour)
		}
	}
	ct.dirty = true
	ct.mu.Unlock()
	ct.Stop()

	ct2 := NewConversationTracker(1*time.Hour, 2*time.Hour, path)
	defer ct2.Stop()

	convs := ct2.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation (expired filtered), got %d", len(convs))
	}
	if convs[0].RawUserID != "user-fresh" {
		t.Errorf("expected user-fresh to survive, got %s", convs[0].RawUserID)
	}
}

func TestConversationTracker_ShortTitleConcatenation(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("messages", "user-x", "model", 0, "ch", "", "驾驶舱对话卡片保存时长", 1)
	ct.UpdateTitle("messages", "user-x", "Hi")

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].Title != "Hi — 驾驶舱对话卡片保存时长" {
		t.Errorf("expected concatenated title, got %q", convs[0].Title)
	}
}

func TestConversationTracker_TitleCompletesWithFallbackUntilLimit(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("messages", "user-x", "model", 0, "ch", "", "怎么给 Go self-update 选择一个合适的实现方案", 1)
	ct.UpdateTitle("messages", "user-x", "Evaluate Go self-update options")

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	want := "Evaluate Go self-update options — 怎么给 Go self-update 选择一个合适的实现方案"
	if convs[0].Title != want {
		t.Errorf("expected completed title %q, got %q", want, convs[0].Title)
	}
}

func TestConversationTracker_UpdatesTitleWhenFallbackChanges(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("messages", "user-x", "model", 0, "ch", "", "第一轮问题", 1)
	ct.UpdateTitle("messages", "user-x", "Evaluate Go self-update options")
	ct.Track("messages", "user-x", "model", 0, "ch", "", "第二轮追问如何自动更新", 2)

	convs := ct.GetActiveConversations("")
	want := "Evaluate Go self-update options — 第二轮追问如何自动更新"
	if convs[0].Title != want {
		t.Errorf("expected title to use latest fallback %q, got %q", want, convs[0].Title)
	}
}

func TestConversationTracker_ShortTitleNoDuplicate(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("messages", "user-x", "model", 0, "ch", "", "驾驶舱对话卡片保存时长", 1)
	ct.UpdateTitle("messages", "user-x", "OK")
	first := ct.GetActiveConversations("")[0].Title
	if first != "OK — 驾驶舱对话卡片保存时长" {
		t.Fatalf("expected first concatenation, got %q", first)
	}
	ct.UpdateTitle("messages", "user-x", "OK")
	second := ct.GetActiveConversations("")[0].Title
	if second != first {
		t.Errorf("expected no duplicate concatenation, got %q (was %q)", second, first)
	}
}

func TestConversationTracker_StopIdempotent(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	ct.Stop()
	ct.Stop()
}

func TestConversationTracker_TrackMultipleRequests(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user123", "claude-sonnet-4-20250514", 0, "primary", "", "", 0)
	ct.Track("chat", "user123", "claude-opus-4-20250514", 1, "backup", "", "", 0)

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation (same user), got %d", len(convs))
	}

	conv := convs[0]
	if conv.RequestCount != 2 {
		t.Errorf("expected requestCount=2, got %d", conv.RequestCount)
	}
	if len(conv.Models) != 2 {
		t.Errorf("expected 2 models, got %d", len(conv.Models))
	}
	if conv.CurrentChannel != 1 {
		t.Errorf("expected currentChannel=1, got %d", conv.CurrentChannel)
	}
}

func TestConversationTracker_DifferentUsers(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user1", "model-a", 0, "ch1", "", "", 0)
	ct.Track("chat", "user2", "model-b", 1, "ch2", "", "", 0)

	convs := ct.GetActiveConversations("")
	if len(convs) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(convs))
	}
}

func TestConversationTracker_KindFilter(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "user1", "model-a", 0, "ch1", "", "", 0)
	ct.Track("messages", "user2", "model-b", 1, "ch2", "", "", 0)

	chatConvs := ct.GetActiveConversations("chat")
	if len(chatConvs) != 1 {
		t.Errorf("expected 1 chat conversation, got %d", len(chatConvs))
	}

	msgConvs := ct.GetActiveConversations("messages")
	if len(msgConvs) != 1 {
		t.Errorf("expected 1 messages conversation, got %d", len(msgConvs))
	}
}

func TestConversationTracker_SessionID(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("responses", "user1", "model-a", 0, "ch1", "sess_abc123", "", 0)
	ct.Track("responses", "user1", "model-a", 0, "ch1", "sess_abc123", "", 0)

	convs := ct.GetActiveConversations("")
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].ID != "sess_abc123" {
		t.Errorf("expected ID=sess_abc123, got %s", convs[0].ID)
	}
	if convs[0].RequestCount != 2 {
		t.Errorf("expected requestCount=2, got %d", convs[0].RequestCount)
	}
}

func TestConversationTracker_EmptyUserID(t *testing.T) {
	ct := NewConversationTracker(1*time.Hour, 2*time.Hour)
	defer ct.Stop()

	ct.Track("chat", "", "model-a", 0, "ch1", "", "", 0)

	convs := ct.GetActiveConversations("")
	if len(convs) != 0 {
		t.Errorf("expected 0 conversations for empty userID, got %d", len(convs))
	}
}

func TestConversationTracker_MaskUserID(t *testing.T) {
	result := maskUserID("short")
	if result != "s***" {
		t.Errorf("expected s***, got %s", result)
	}

	result = maskUserID("longUserIdentifier")
	if result != "long***fier" {
		t.Errorf("expected long***fier, got %s", result)
	}

	result = maskUserID("user_abc123_session_dbf5ffc0-dea5-44ca")
	if result != "sess:dbf5ffc0" {
		t.Errorf("expected sess:dbf5ffc0, got %s", result)
	}

	result = maskUserID("a_very_long_user_id_that_has_no_sess_keyword_1234567890")
	if result != "a_very_l...7890" {
		t.Errorf("expected a_very_l...7890, got %s", result)
	}
}
