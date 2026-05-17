package responses

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/gin-gonic/gin"
)

func TestFormatItemsAsTranscript_BasicMessages(t *testing.T) {
	items := []types.ResponsesItem{
		{Type: "message", Role: "user", Content: "Hello world"},
		{Type: "message", Role: "assistant", Content: "Hi there"},
	}
	transcript := formatItemsAsTranscript(items)
	if !strings.Contains(transcript, "[User]\nHello world") {
		t.Fatalf("missing user message in transcript: %s", transcript)
	}
	if !strings.Contains(transcript, "[Assistant]\nHi there") {
		t.Fatalf("missing assistant message in transcript: %s", transcript)
	}
	if !strings.Contains(transcript, "---") {
		t.Fatalf("missing separator in transcript: %s", transcript)
	}
}

func TestFormatItemsAsTranscript_FunctionCall(t *testing.T) {
	items := []types.ResponsesItem{
		{Type: "function_call", Name: "Read", Arguments: `{"file":"/tmp/x"}`},
	}
	transcript := formatItemsAsTranscript(items)
	if !strings.Contains(transcript, "Tool Call: Read") {
		t.Fatalf("missing tool call in transcript: %s", transcript)
	}
}

// PLACEHOLDER_MORE_TESTS

func TestFormatItemsAsTranscript_SkipsFunctionCallOutput(t *testing.T) {
	items := []types.ResponsesItem{
		{Type: "function_call_output", Output: "very long output"},
	}
	transcript := formatItemsAsTranscript(items)
	if transcript != "" {
		t.Fatalf("function_call_output should be skipped, got: %s", transcript)
	}
}

func TestFormatItemsAsTranscript_ContentBlocks(t *testing.T) {
	items := []types.ResponsesItem{
		{Type: "message", Role: "user", Content: []interface{}{
			map[string]interface{}{"type": "input_text", "text": "block1"},
			map[string]interface{}{"type": "input_text", "text": "block2"},
		}},
	}
	transcript := formatItemsAsTranscript(items)
	if !strings.Contains(transcript, "block1") || !strings.Contains(transcript, "block2") {
		t.Fatalf("missing content blocks in transcript: %s", transcript)
	}
}

func TestTruncateTranscript(t *testing.T) {
	long := strings.Repeat("a", localCompactMaxTranscriptRunes+1000)
	result := truncateTranscript(long)
	if len([]rune(result)) >= len([]rune(long)) {
		t.Fatalf("transcript should be truncated")
	}
	if !strings.Contains(result, "omitted") {
		t.Fatalf("truncated transcript should contain omitted marker")
	}
}

func TestNeedsLocalCompact(t *testing.T) {
	tests := []struct {
		serviceType string
		want        bool
	}{
		{"responses", false},
		{"openai", true},
		{"claude", true},
		{"gemini", true},
		{"", true},
	}
	for _, tt := range tests {
		got := needsLocalCompact(&config.UpstreamConfig{ServiceType: tt.serviceType})
		if got != tt.want {
			t.Errorf("needsLocalCompact(%q) = %v, want %v", tt.serviceType, got, tt.want)
		}
	}
}

func TestIsNativeCompactUnsupported(t *testing.T) {
	if !isNativeCompactUnsupported(404) {
		t.Error("404 should be unsupported")
	}
	if !isNativeCompactUnsupported(405) {
		t.Error("405 should be unsupported")
	}
	if isNativeCompactUnsupported(401) {
		t.Error("401 should not be unsupported")
	}
}

func TestLocalCompact_OpenAIUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		// 验证请求被转换为 chat completions 格式
		if _, ok := req["messages"]; !ok {
			t.Error("expected messages field in converted request")
		}
		if req["stream"] != false {
			t.Error("expected stream=false for non-streaming compact")
		}
		// 不应有 tools
		if _, ok := req["tools"]; ok {
			t.Error("compact request should not have tools")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		io.WriteString(w, `{"id":"chatcmpl-123","choices":[{"message":{"role":"assistant","content":"## Summary\nCompacted context"}}],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`)
	}))
	defer upstream.Close()

	sessionManager := session.NewSessionManager(24*60*60*1000000000, 100, 100000)

	cfgManager := setupResponsesTestConfigManager(t, []config.UpstreamConfig{{
		Name:        "openai-channel",
		BaseURL:     upstream.URL,
		APIKeys:     []string{"sk-test"},
		ServiceType: "openai",
		Status:      "active",
	}})

	messagesMetrics := metrics.NewMetricsManager()
	responsesMetrics := metrics.NewMetricsManager()
	geminiMetrics := metrics.NewMetricsManager()
	chatMetrics := metrics.NewMetricsManager()
	imagesMetrics := metrics.NewMetricsManager()
	traceAffinity := session.NewTraceAffinityManager()
	t.Cleanup(func() {
		messagesMetrics.Stop()
		responsesMetrics.Stop()
		geminiMetrics.Stop()
		chatMetrics.Stop()
		imagesMetrics.Stop()
		traceAffinity.Stop()
	})

	sch := scheduler.NewChannelScheduler(cfgManager, messagesMetrics, responsesMetrics, geminiMetrics, chatMetrics, imagesMetrics, traceAffinity, nil)

	envCfg := &config.EnvConfig{
		ProxyAccessKey:     "secret-key",
		MaxRequestBodySize: 1024 * 1024,
	}

	r := gin.New()
	r.POST("/v1/responses/compact", CompactHandler(envCfg, cfgManager, sessionManager, sch))

	body := `{"model":"gpt-4o","input":[{"type":"message","role":"user","content":"Hello"}],"stream":false}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var resp types.ResponsesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v, body=%s", err, w.Body.String())
	}
	if resp.Status != "completed" {
		t.Fatalf("status = %q, want completed", resp.Status)
	}
	if len(resp.Output) == 0 {
		t.Fatal("expected output items")
	}
}

func TestGetSessionByResponseID(t *testing.T) {
	sm := session.NewSessionManager(24*60*60*1000000000, 100, 100000)

	// 未命中
	_, err := sm.GetSessionByResponseID("resp_nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent response ID")
	}

	// 创建 session 并记录映射
	sess, _ := sm.GetOrCreateSession("")
	sm.RecordResponseMapping("resp_123", sess.ID)

	// 命中
	found, err := sm.GetSessionByResponseID("resp_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != sess.ID {
		t.Fatalf("session ID = %q, want %q", found.ID, sess.ID)
	}
}
