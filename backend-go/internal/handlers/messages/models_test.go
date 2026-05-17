package messages

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/metrics"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/gin-gonic/gin"
)

func setupModelsConfigManager(t *testing.T, cfg config.Config) *config.ConfigManager {
	t.Helper()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	cm, err := config.NewConfigManager(tmpFile)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}
	t.Cleanup(func() { _ = cm.Close() })
	return cm
}

func newModelsTestScheduler(cfgManager *config.ConfigManager) *scheduler.ChannelScheduler {
	traceAffinity := session.NewTraceAffinityManager()
	metricsManagers := []*metrics.MetricsManager{
		metrics.NewMetricsManager(),
		metrics.NewMetricsManager(),
		metrics.NewMetricsManager(),
		metrics.NewMetricsManager(),
		metrics.NewMetricsManager(),
	}

	schedulerInstance := scheduler.NewChannelScheduler(
		cfgManager,
		metricsManagers[0],
		metricsManagers[1],
		metricsManagers[2],
		metricsManagers[3],
		metricsManagers[4],
		traceAffinity,
		nil,
	)

	return schedulerInstance
}

func newModelsRouterForAggregate(envCfg *config.EnvConfig, cfgManager *config.ConfigManager, sch *scheduler.ChannelScheduler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/models", ModelsHandler(envCfg, cfgManager, sch))
	r.GET("/:routePrefix/v1/models", ModelsHandler(envCfg, cfgManager, sch))
	r.GET("/v1/models/:model", ModelsDetailHandler(envCfg, cfgManager, sch))
	r.GET("/:routePrefix/v1/models/:model", ModelsDetailHandler(envCfg, cfgManager, sch))
	return r
}

func TestModelsHandler_UsesActiveKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-active" {
			t.Fatalf("Authorization = %q, want active key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-active","object":"model"}]}`))
	}))
	defer upstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "messages-active",
			BaseURL:     upstream.URL,
			APIKeys:     []string{"sk-active"},
			ServiceType: "claude",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body == "" || body == "{}" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestModelsHandler_FallbackToDisabledKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-disabled" {
			t.Fatalf("Authorization = %q, want disabled fallback key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-disabled","object":"model"}]}`))
	}))
	defer upstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:    "messages-disabled-fallback",
			BaseURL: upstream.URL,
			DisabledAPIKeys: []config.DisabledKeyInfo{{
				Key:        "sk-disabled",
				Reason:     "authentication_error",
				Message:    "invalid key",
				DisabledAt: "2026-04-15T00:00:00Z",
			}},
			ServiceType: "claude",
			Status:      "active",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body == "" || body == "{}" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestModelsHandler_FallbackToDisabledKeyRespectsRoutePrefix(t *testing.T) {
	matchedUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-prefix" {
			t.Fatalf("Authorization = %q, want prefixed disabled fallback key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-prefix","object":"model"}]}`))
	}))
	defer matchedUpstream.Close()

	defaultUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("default route fallback should not be used for prefixed request")
	}))
	defer defaultUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "default-disabled",
				BaseURL: defaultUpstream.URL,
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-default",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "claude",
				Status:      "active",
			},
			{
				Name:        "prefixed-disabled",
				BaseURL:     matchedUpstream.URL,
				RoutePrefix: "kimi",
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-prefix",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "claude",
				Status:      "active",
			},
		},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/kimi/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestModelsHandler_FallbackToDisabledKeySkipsDisabledChannels(t *testing.T) {
	disabledUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("disabled channel should not be used for fallback")
	}))
	defer disabledUpstream.Close()

	activeFallbackUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-active-disabled" {
			t.Fatalf("Authorization = %q, want active-channel disabled fallback key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-active-disabled","object":"model"}]}`))
	}))
	defer activeFallbackUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "explicitly-disabled",
				BaseURL: disabledUpstream.URL,
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-disabled-channel",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "claude",
				Status:      "disabled",
			},
			{
				Name:    "active-with-disabled-keys",
				BaseURL: activeFallbackUpstream.URL,
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-active-disabled",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "claude",
				Status:      "active",
			},
		},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestModelsHandler_NoKeysStillFails(t *testing.T) {
	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "messages-no-keys",
			BaseURL:     "https://example.com",
			ServiceType: "claude",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil).WithContext(context.Background())
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestModelsHandler_MergesChatModels(t *testing.T) {
	messagesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-messages","object":"model"},{"id":"model-shared","object":"model"}]}`))
	}))
	defer messagesUpstream.Close()

	responsesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-responses","object":"model"},{"id":"model-shared","object":"model"}]}`))
	}))
	defer responsesUpstream.Close()

	chatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-chat","object":"model"},{"id":"model-shared","object":"model"}]}`))
	}))
	defer chatUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "messages-active",
			BaseURL:     messagesUpstream.URL,
			APIKeys:     []string{"sk-messages"},
			ServiceType: "claude",
		}},
		ResponsesUpstream: []config.UpstreamConfig{{
			Name:        "responses-active",
			BaseURL:     responsesUpstream.URL,
			APIKeys:     []string{"sk-responses"},
			ServiceType: "responses",
		}},
		ChatUpstream: []config.UpstreamConfig{{
			Name:        "chat-active",
			BaseURL:     chatUpstream.URL,
			APIKeys:     []string{"sk-chat"},
			ServiceType: "openai",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}

	var resp ModelsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	ids := make([]string, 0, len(resp.Data))
	for _, model := range resp.Data {
		ids = append(ids, model.ID)
	}

	want := []string{"model-messages", "model-shared", "model-responses", "model-chat"}
	if len(ids) != len(want) {
		t.Fatalf("ids len = %d, want %d, ids=%v", len(ids), len(want), ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids[%d] = %q, want %q, ids=%v", i, ids[i], want[i], ids)
		}
	}
}

func TestModelsDetailHandler_FallsBackToChat(t *testing.T) {
	messagesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer messagesUpstream.Close()

	responsesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer responsesUpstream.Close()

	chatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models/model-chat" {
			t.Fatalf("path = %q, want /v1/models/model-chat", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"model-chat","object":"model","owned_by":"chat"}`))
	}))
	defer chatUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "messages-active",
			BaseURL:     messagesUpstream.URL,
			APIKeys:     []string{"sk-messages"},
			ServiceType: "claude",
		}},
		ResponsesUpstream: []config.UpstreamConfig{{
			Name:        "responses-active",
			BaseURL:     responsesUpstream.URL,
			APIKeys:     []string{"sk-responses"},
			ServiceType: "responses",
		}},
		ChatUpstream: []config.UpstreamConfig{{
			Name:        "chat-active",
			BaseURL:     chatUpstream.URL,
			APIKeys:     []string{"sk-chat"},
			ServiceType: "openai",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models/model-chat", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != `{"id":"model-chat","object":"model","owned_by":"chat"}` {
		t.Fatalf("body = %s", got)
	}
}

func TestModelsDetailHandler_PrefersMessagesOverChat(t *testing.T) {
	messagesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"model-shared","object":"model","owned_by":"messages"}`))
	}))
	defer messagesUpstream.Close()

	responsesUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"model-shared","object":"model","owned_by":"responses"}`))
	}))
	defer responsesUpstream.Close()

	chatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"model-shared","object":"model","owned_by":"chat"}`))
	}))
	defer chatUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{{
			Name:        "messages-active",
			BaseURL:     messagesUpstream.URL,
			APIKeys:     []string{"sk-messages"},
			ServiceType: "claude",
		}},
		ResponsesUpstream: []config.UpstreamConfig{{
			Name:        "responses-active",
			BaseURL:     responsesUpstream.URL,
			APIKeys:     []string{"sk-responses"},
			ServiceType: "responses",
		}},
		ChatUpstream: []config.UpstreamConfig{{
			Name:        "chat-active",
			BaseURL:     chatUpstream.URL,
			APIKeys:     []string{"sk-chat"},
			ServiceType: "openai",
		}},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models/model-shared", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != `{"id":"model-shared","object":"model","owned_by":"messages"}` {
		t.Fatalf("body = %s", got)
	}
}

func TestModelsDetailHandler_ChatFallbackRespectsRoutePrefix(t *testing.T) {
	defaultChatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("default route chat fallback should not be used for prefixed request")
	}))
	defer defaultChatUpstream.Close()

	prefixedChatUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-prefix-chat" {
			t.Fatalf("Authorization = %q, want prefixed chat disabled fallback key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"model-prefix","object":"model","owned_by":"chat"}`))
	}))
	defer prefixedChatUpstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		ChatUpstream: []config.UpstreamConfig{
			{
				Name:    "default-chat-disabled",
				BaseURL: defaultChatUpstream.URL,
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-default-chat",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "openai",
				Status:      "active",
			},
			{
				Name:        "prefixed-chat-disabled",
				BaseURL:     prefixedChatUpstream.URL,
				RoutePrefix: "kimi",
				DisabledAPIKeys: []config.DisabledKeyInfo{{
					Key:        "sk-prefix-chat",
					Reason:     "authentication_error",
					Message:    "invalid key",
					DisabledAt: "2026-04-15T00:00:00Z",
				}},
				ServiceType: "openai",
				Status:      "active",
			},
		},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/kimi/v1/models/model-prefix", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestBuildClaudeCompatibleModelsURLs(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected []string
	}{
		{
			name:    "纯域名不产生额外候选",
			baseURL: "https://api.anthropic.com",
			expected: []string{
				"https://api.anthropic.com/v1/models",
			},
		},
		{
			name:    "带 /anthropic 尾段产生两个候选（剔除后即纯域名）",
			baseURL: "https://api.deepseek.com/anthropic",
			expected: []string{
				"https://api.deepseek.com/anthropic/v1/models",
				"https://api.deepseek.com/v1/models",
			},
		},
		{
			name:    "带 /anthropic/v1 尾段产生两个候选",
			baseURL: "https://api.deepseek.com/anthropic/v1",
			expected: []string{
				"https://api.deepseek.com/anthropic/v1/models",
				"https://api.deepseek.com/v1/models",
			},
		},
		{
			name:    "带 /proxy/anthropic 产生三个候选",
			baseURL: "https://api.vendor.com/proxy/anthropic",
			expected: []string{
				"https://api.vendor.com/proxy/anthropic/v1/models",
				"https://api.vendor.com/proxy/v1/models",
				"https://api.vendor.com/v1/models",
			},
		},
		{
			name:    "带 /proxy/claude/v1 产生三个候选",
			baseURL: "https://api.vendor.com/proxy/claude/v1",
			expected: []string{
				"https://api.vendor.com/proxy/claude/v1/models",
				"https://api.vendor.com/proxy/v1/models",
				"https://api.vendor.com/v1/models",
			},
		},
		{
			name:    "带 /messages 尾段产生两个候选",
			baseURL: "https://api.vendor.com/messages",
			expected: []string{
				"https://api.vendor.com/messages/v1/models",
				"https://api.vendor.com/v1/models",
			},
		},
		{
			name:    "非协议尾段不产生额外候选",
			baseURL: "https://api.vendor.com/openai",
			expected: []string{
				"https://api.vendor.com/openai/v1/models",
			},
		},
		{
			name:    "# 标记保持兼容",
			baseURL: "https://api.vendor.com/anthropic#",
			expected: []string{
				"https://api.vendor.com/anthropic/models",
				"https://api.vendor.com/v1/models",
			},
		},
		{
			name:    "带端口的域名",
			baseURL: "https://localhost:8080/anthropic",
			expected: []string{
				"https://localhost:8080/anthropic/v1/models",
				"https://localhost:8080/v1/models",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClaudeCompatibleModelsURLs(tt.baseURL)
			if len(got) != len(tt.expected) {
				t.Fatalf("候选数量不匹配: got %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("候选[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTryModelsRequest_ClaudeCompatFallback(t *testing.T) {
	// 模拟上游：第一个 URL 返回 404，第二个返回 200
	callCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/anthropic/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"deepseek-chat","object":"model","owned_by":"deepseek"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	cfgManager := setupModelsConfigManager(t, config.Config{
		Upstream: []config.UpstreamConfig{
			{
				Name:    "deepseek-compat",
				BaseURL: upstream.URL + "/anthropic",
				APIKeys: []string{"sk-test"},
				Status:  "active",
			},
		},
	})
	sch := newModelsTestScheduler(cfgManager)
	router := newModelsRouterForAggregate(&config.EnvConfig{ProxyAccessKey: "test-key"}, cfgManager, sch)

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if callCount < 2 {
		t.Errorf("期望至少 2 次请求（第一次 404 后 fallback），实际 %d 次", callCount)
	}

	var resp ModelsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("期望返回模型列表，但为空")
	}
	if resp.Data[0].ID != "deepseek-chat" {
		t.Errorf("模型 ID = %q, want %q", resp.Data[0].ID, "deepseek-chat")
	}
}
