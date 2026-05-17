// Package messages 提供 Claude Messages API 的处理器
package messages

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/handlers/common"
	"github.com/BenedictKing/ccx/internal/middleware"
	"github.com/BenedictKing/ccx/internal/providers"
	"github.com/BenedictKing/ccx/internal/scheduler"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

// Handler Messages API 代理处理器
// 支持多渠道调度：当配置多个渠道时自动启用
func Handler(envCfg *config.EnvConfig, cfgManager *config.ConfigManager, channelScheduler *scheduler.ChannelScheduler) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 先进行认证
		middleware.ProxyAuthMiddleware(envCfg)(c)
		if c.IsAborted() {
			return
		}

		startTime := time.Now()

		// 读取请求体
		bodyBytes, err := common.ReadRequestBody(c, envCfg.MaxRequestBodySize)
		if err != nil {
			return
		}

		// 预处理：移除空 signature 字段，预防 400 错误
		// modified 表示请求体是否被修改，详细日志由 RemoveEmptySignatures 内部记录
		bodyBytes, modified := common.RemoveEmptySignatures(bodyBytes, envCfg.EnableRequestLogs, "Messages")
		_ = modified // 保留以便未来扩展（如需在 handler 层面做额外处理）

		// 预处理：清理历史 thinking 内容块/字段，预防上游参数校验 400
		bodyBytes, thinkingModified := common.SanitizeMalformedThinkingBlocks(bodyBytes, envCfg.EnableRequestLogs, "Messages")
		_ = thinkingModified // 保留以便未来扩展（如需在 handler 层面做额外处理）

		// 预处理：移除 system 中的 cch= 计费参数
		if cfgManager.GetStripBillingHeader() {
			bodyBytes, _ = common.RemoveBillingHeaders(bodyBytes, envCfg.EnableRequestLogs, "Messages")
		}

		// 入口保留原始请求体；按渠道在发往上游前决定是否规范化 metadata.user_id
		c.Set("requestBodyBytes", bodyBytes)

		// 解析请求
		var claudeReq types.ClaudeRequest
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &claudeReq)
		}

		// 提取统一会话标识用于 Trace 亲和性（保持 metadata.user_id 默认规范化后的既有路由语义）
		affinityBody := common.NormalizeMetadataUserID(bodyBytes)
		userID := utils.ExtractUnifiedSessionID(c, affinityBody)

		isTitleRequest := isClaudeCodeTitleRequest(bodyBytes)
		if envCfg.ShouldLog("debug") && isTitleRequest {
			log.Printf("[Messages-Title-Debug] 检测到 Claude Code title 请求: user=%s, model=%s, stream=%t",
				scheduler.MaskUserIDForLog(userID), claudeReq.Model, claudeReq.Stream)
		}

		// 提取用户最后一条消息用于对话标题 fallback
		if !isTitleRequest {
			c.Set("lastUserMessage", extractLastUserMessage(claudeReq.Messages))
			c.Set("userMessageCount", countUserMessages(claudeReq.Messages))
		}

		// 记录原始请求信息（仅在入口处记录一次）
		common.LogOriginalRequest(c, bodyBytes, envCfg, "Messages")

		// 检查是否为多渠道模式
		isMultiChannel := channelScheduler.IsMultiChannelMode(scheduler.ChannelKindMessages)

		if isMultiChannel {
			handleMultiChannel(c, envCfg, cfgManager, channelScheduler, bodyBytes, claudeReq, userID, startTime)
		} else {
			handleSingleChannel(c, envCfg, cfgManager, channelScheduler, bodyBytes, claudeReq, startTime)
		}
	})
}

// handleMultiChannel 处理多渠道代理请求
func handleMultiChannel(
	c *gin.Context,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
	bodyBytes []byte,
	claudeReq types.ClaudeRequest,
	userID string,
	startTime time.Time,
) {
	isTitleRequest := isClaudeCodeTitleRequest(bodyBytes)

	common.HandleMultiChannelFailover(
		c,
		envCfg,
		channelScheduler,
		scheduler.ChannelKindMessages,
		"Messages",
		userID,
		claudeReq.Model,
		func(selection *scheduler.SelectionResult) common.MultiChannelAttemptResult {
			upstream := selection.Upstream
			channelIndex := selection.ChannelIndex

			if upstream == nil {
				return common.MultiChannelAttemptResult{}
			}

			provider := providers.GetProvider(upstream.ServiceType)
			if provider == nil {
				return common.MultiChannelAttemptResult{}
			}

			metricsManager := channelScheduler.GetMessagesMetricsManager()
			baseURLs := upstream.GetAllBaseURLs()
			sortedURLResults := channelScheduler.GetSortedURLsForChannel(scheduler.ChannelKindMessages, channelIndex, baseURLs)

			handled, successKey, successBaseURLIdx, failoverErr, usage, lastErr := common.TryUpstreamWithAllKeys(
				c,
				envCfg,
				cfgManager,
				channelScheduler,
				scheduler.ChannelKindMessages,
				"Messages",
				metricsManager,
				upstream,
				sortedURLResults,
				bodyBytes,
				claudeReq.Stream,
				func(upstream *config.UpstreamConfig, failedKeys map[string]bool) (string, error) {
					return cfgManager.GetNextAPIKey(upstream, failedKeys, "Messages")
				},
				func(c *gin.Context, upstreamCopy *config.UpstreamConfig, apiKey string) (*http.Request, error) {
					req, _, err := provider.ConvertToProviderRequest(c, upstreamCopy, apiKey)
					return req, err
				},
				func(apiKey string) {
					if err := cfgManager.DeprioritizeAPIKey(apiKey); err != nil {
						log.Printf("[Messages-Key] 警告: 密钥降级失败: %v", err)
					}
				},
				func(url string) {
					channelScheduler.MarkURLFailure(scheduler.ChannelKindMessages, channelIndex, url)
				},
				func(url string) {
					channelScheduler.MarkURLSuccess(scheduler.ChannelKindMessages, channelIndex, url)
				},
				func(c *gin.Context, resp *http.Response, upstreamCopy *config.UpstreamConfig, apiKey string, actualRequestBody []byte) (*types.Usage, error) {
					if claudeReq.Stream {
						return common.HandleStreamResponse(c, resp, provider, envCfg, startTime, upstreamCopy, actualRequestBody, claudeReq.Model)
					}
					return handleNormalResponse(c, resp, provider, envCfg, startTime, actualRequestBody, upstreamCopy, apiKey, cfgManager.GetFuzzyModeEnabled())
				},
				claudeReq.Model,
				"",
				selection.ChannelIndex,
				channelScheduler.GetChannelLogStore(scheduler.ChannelKindMessages),
			)

			responseText, _ := c.Get("responseText")
			return common.MultiChannelAttemptResult{
				Handled:           handled,
				Attempted:         true,
				SuccessKey:        successKey,
				SuccessBaseURLIdx: successBaseURLIdx,
				FailoverError:     failoverErr,
				Usage:             usage,
				LastError:         lastErr,
				ResponseText:      responseTextString(responseText),
			}
		},
		func(selection *scheduler.SelectionResult, result common.MultiChannelAttemptResult) {
			if !isTitleRequest || result.ResponseText == "" {
				return
			}
			title := extractTitleFromResponseText(result.ResponseText)
			updated := channelScheduler.UpdateConversationTitle(scheduler.ChannelKindMessages, userID, title)
			if envCfg.ShouldLog("debug") {
				log.Printf("[Messages-Title-Debug] title 更新结果: user=%s, title=%q, updated=%t, responseTextLen=%d",
					scheduler.MaskUserIDForLog(userID), title, updated, len(result.ResponseText))
			}
		},
		func(ctx *gin.Context, failoverErr *common.FailoverError, lastError error) {
			common.HandleAllChannelsFailed(ctx, cfgManager.GetFuzzyModeEnabled(), failoverErr, lastError, "Messages")
		},
	)
}

// handleSingleChannel 处理单渠道代理请求
func handleSingleChannel(
	c *gin.Context,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	channelScheduler *scheduler.ChannelScheduler,
	bodyBytes []byte,
	claudeReq types.ClaudeRequest,
	startTime time.Time,
) {
	upstream, channelIndex, err := cfgManager.GetCurrentUpstreamWithIndex()
	if err != nil {
		c.JSON(503, gin.H{
			"error": "未配置任何渠道，请先在管理界面添加渠道",
			"code":  "NO_UPSTREAM",
		})
		return
	}

	if len(upstream.APIKeys) == 0 {
		c.JSON(503, gin.H{
			"error": fmt.Sprintf("当前渠道 \"%s\" 未配置API密钥", upstream.Name),
			"code":  "NO_API_KEYS",
		})
		return
	}

	provider := providers.GetProvider(upstream.ServiceType)
	if provider == nil {
		c.JSON(400, gin.H{"error": "Unsupported service type"})
		return
	}

	metricsManager := channelScheduler.GetMessagesMetricsManager()
	baseURLs := upstream.GetAllBaseURLs()

	urlResults := common.BuildDefaultURLResults(baseURLs)

	handled, _, _, lastFailoverError, _, lastError := common.TryUpstreamWithAllKeys(
		c,
		envCfg,
		cfgManager,
		channelScheduler,
		scheduler.ChannelKindMessages,
		"Messages",
		metricsManager,
		upstream,
		urlResults,
		bodyBytes,
		claudeReq.Stream,
		func(upstream *config.UpstreamConfig, failedKeys map[string]bool) (string, error) {
			return cfgManager.GetNextAPIKey(upstream, failedKeys, "Messages")
		},
		func(c *gin.Context, upstreamCopy *config.UpstreamConfig, apiKey string) (*http.Request, error) {
			req, _, err := provider.ConvertToProviderRequest(c, upstreamCopy, apiKey)
			return req, err
		},
		func(apiKey string) {
			if err := cfgManager.DeprioritizeAPIKey(apiKey); err != nil {
				log.Printf("[Messages-Key] 警告: 密钥降级失败: %v", err)
			}
		},
		nil,
		nil,
		func(c *gin.Context, resp *http.Response, upstreamCopy *config.UpstreamConfig, apiKey string, actualRequestBody []byte) (*types.Usage, error) {
			if claudeReq.Stream {
				return common.HandleStreamResponse(c, resp, provider, envCfg, startTime, upstreamCopy, actualRequestBody, claudeReq.Model)
			}
			return handleNormalResponse(c, resp, provider, envCfg, startTime, actualRequestBody, upstreamCopy, apiKey, cfgManager.GetFuzzyModeEnabled())
		},
		claudeReq.Model,
		"",
		channelIndex,
		channelScheduler.GetChannelLogStore(scheduler.ChannelKindMessages),
	)
	if handled {
		userID := utils.ExtractUnifiedSessionID(c, common.NormalizeMetadataUserID(bodyBytes))
		isTitleRequest := isClaudeCodeTitleRequest(bodyBytes)
		if !isTitleRequest {
			lastUserMessage := extractLastUserMessage(claudeReq.Messages)
			userMessageCount := countUserMessages(claudeReq.Messages)
			channelScheduler.SetTraceAffinity(userID, channelIndex, scheduler.ChannelKindMessages)
			channelScheduler.TrackConversation(scheduler.ChannelKindMessages, userID, claudeReq.Model, channelIndex, upstream.Name, "", lastUserMessage, userMessageCount)
			if envCfg.ShouldLog("debug") {
				log.Printf("[Messages-Conversation-Debug] 已追踪单渠道对话: user=%s, model=%s, channel=%d, userMessages=%d, hasFallbackTitle=%t",
					scheduler.MaskUserIDForLog(userID), claudeReq.Model, channelIndex, userMessageCount, lastUserMessage != "")
			}
		} else {
			responseText, _ := c.Get("responseText")
			title := extractTitleFromResponseText(responseTextString(responseText))
			updated := channelScheduler.UpdateConversationTitle(scheduler.ChannelKindMessages, userID, title)
			if envCfg.ShouldLog("debug") {
				log.Printf("[Messages-Title-Debug] 单渠道 title 更新结果: user=%s, title=%q, updated=%t, responseTextLen=%d",
					scheduler.MaskUserIDForLog(userID), title, updated, len(responseTextString(responseText)))
			}
		}
		return
	}

	log.Printf("[Messages-Error] 所有API密钥都失败了")
	common.HandleAllKeysFailed(c, cfgManager.GetFuzzyModeEnabled(), lastFailoverError, lastError, "Messages")
}

// handleNormalResponse 处理非流式响应
func handleNormalResponse(
	c *gin.Context,
	resp *http.Response,
	provider providers.Provider,
	envCfg *config.EnvConfig,
	startTime time.Time,
	requestBody []byte,
	upstream *config.UpstreamConfig,
	apiKey string,
	fuzzyMode bool,
) (*types.Usage, error) {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read response"})
		return nil, err
	}

	if envCfg.EnableResponseLogs {
		responseTime := time.Since(startTime).Milliseconds()
		log.Printf("[Messages-Timing] 响应完成: %dms, 状态: %d", responseTime, resp.StatusCode)
		common.LogUpstreamResponse(resp, bodyBytes, envCfg, "Messages")
	}

	providerResp := &types.ProviderResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyBytes,
		Stream:     false,
	}

	claudeResp, err := provider.ConvertToClaudeResponse(providerResp)
	if err != nil {
		// JSON 解析失败（如上游返回 HTML 错误页面）：不写 Header，返回可 failover 的错误
		preview := bodyBytes
		if len(preview) > 100 {
			preview = preview[:100]
		}
		log.Printf("[Messages-InvalidBody] 响应体解析失败: %v, body前100字节: %s", err, preview)
		return nil, fmt.Errorf("%w: %v", common.ErrInvalidResponseBody, err)
	}

	// 空响应拦截（仅 Fuzzy 模式）：上游 200 但 content 语义为空，
	// Header 未发送，可安全 failover 到下一个 Key/BaseURL/渠道
	if fuzzyMode && common.IsClaudeResponseEmpty(claudeResp) {
		log.Printf("[Messages-EmptyResponse] 上游返回空响应（非流式，Key: %s），触发 failover", utils.MaskAPIKey(apiKey))
		return nil, common.ErrEmptyNonStreamResponse
	}

	// Token 补全逻辑
	if claudeResp.Usage == nil {
		estimatedInput := utils.EstimateRequestTokens(requestBody)
		estimatedOutput := utils.EstimateResponseTokens(claudeResp.Content)
		claudeResp.Usage = &types.Usage{
			InputTokens:  estimatedInput,
			OutputTokens: estimatedOutput,
		}
		if envCfg.EnableResponseLogs {
			log.Printf("[Messages-Token] 上游无Usage, 本地估算: input=%d, output=%d", estimatedInput, estimatedOutput)
		}
	} else {
		originalInput := claudeResp.Usage.InputTokens
		originalOutput := claudeResp.Usage.OutputTokens
		patched := false

		hasCacheTokens := claudeResp.Usage.CacheCreationInputTokens > 0 || claudeResp.Usage.CacheReadInputTokens > 0

		if claudeResp.Usage.InputTokens <= 1 && !hasCacheTokens {
			claudeResp.Usage.InputTokens = utils.EstimateRequestTokens(requestBody)
			patched = true
		}
		if claudeResp.Usage.OutputTokens <= 1 {
			claudeResp.Usage.OutputTokens = utils.EstimateResponseTokens(claudeResp.Content)
			patched = true
		}
		if envCfg.EnableResponseLogs {
			if patched {
				log.Printf("[Messages-Token] 虚假值补全: InputTokens=%d->%d, OutputTokens=%d->%d",
					originalInput, claudeResp.Usage.InputTokens, originalOutput, claudeResp.Usage.OutputTokens)
			}
			log.Printf("[Messages-Token] InputTokens=%d, OutputTokens=%d, CacheCreationInputTokens=%d, CacheReadInputTokens=%d, CacheCreation5m=%d, CacheCreation1h=%d, CacheTTL=%s",
				claudeResp.Usage.InputTokens, claudeResp.Usage.OutputTokens,
				claudeResp.Usage.CacheCreationInputTokens, claudeResp.Usage.CacheReadInputTokens,
				claudeResp.Usage.CacheCreation5mInputTokens, claudeResp.Usage.CacheCreation1hInputTokens,
				claudeResp.Usage.CacheTTL)
		}
	}

	// 监听客户端断开连接
	ctx := c.Request.Context()
	go func() {
		<-ctx.Done()
		if !c.Writer.Written() {
			if envCfg.EnableResponseLogs {
				responseTime := time.Since(startTime).Milliseconds()
				log.Printf("[Messages-Timing] 响应中断: %dms, 状态: %d", responseTime, resp.StatusCode)
			}
		}
	}()

	// 转发上游响应头
	utils.ForwardResponseHeaders(resp.Header, c.Writer)

	c.JSON(200, claudeResp)

	if envCfg.EnableResponseLogs {
		responseTime := time.Since(startTime).Milliseconds()
		log.Printf("[Messages-Timing] 响应发送完成: %dms, 状态: %d", responseTime, resp.StatusCode)
	}

	return claudeResp.Usage, nil
}

func isClaudeCodeTitleRequest(bodyBytes []byte) bool {
	var req struct {
		OutputConfig struct {
			Format struct {
				Schema struct {
					Required []string `json:"required"`
				} `json:"schema"`
			} `json:"format"`
		} `json:"output_config"`
		System []struct {
			Text string `json:"text"`
		} `json:"system"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		return false
	}

	requiresTitle := false
	for _, field := range req.OutputConfig.Format.Schema.Required {
		if field == "title" {
			requiresTitle = true
			break
		}
	}
	if !requiresTitle {
		return false
	}

	for _, block := range req.System {
		if strings.Contains(block.Text, "Generate a concise") && strings.Contains(block.Text, "title") {
			return true
		}
	}
	return false
}

func extractTitleFromResponseText(responseText string) string {
	responseText = strings.TrimSpace(responseText)
	if responseText == "" {
		return ""
	}

	var payload struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(responseText), &payload); err == nil {
		return strings.TrimSpace(payload.Title)
	}

	return strings.Trim(strings.TrimSpace(responseText), `"`)
}

func responseTextString(value interface{}) string {
	text, _ := value.(string)
	return text
}

func countUserMessages(messages []types.ClaudeMessage) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == "user" && len(extractUserTextBlocks(msg)) > 0 {
			count++
		}
	}
	return count
}

func extractLastUserMessage(messages []types.ClaudeMessage) string {
	const maxLen = 80
	var parts []string
	totalLen := 0

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		texts := extractUserTextBlocks(messages[i])
		for j := len(texts) - 1; j >= 0; j-- {
			parts = append(parts, texts[j])
			totalLen += len([]rune(texts[j]))
			if totalLen >= maxLen {
				break
			}
		}
		if totalLen >= maxLen {
			break
		}
	}

	if len(parts) == 0 {
		return ""
	}

	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " / ")
}

func extractUserTextBlocks(message types.ClaudeMessage) []string {
	texts := []string{}
	appendText := func(text string) {
		if cleaned := cleanUserTitleText(text); cleaned != "" {
			texts = append(texts, cleaned)
		}
	}

	switch content := message.Content.(type) {
	case string:
		appendText(content)
	case []interface{}:
		for _, block := range content {
			m, ok := block.(map[string]interface{})
			if !ok || m["type"] != "text" {
				continue
			}
			if text, ok := m["text"].(string); ok {
				appendText(text)
			}
		}
	}
	return texts
}

func cleanUserTitleText(text string) string {
	text = removeTaggedBlocks(text, "system-reminder")
	text = removeTaggedBlocks(text, "local-command-caveat")
	text = removeTaggedBlocks(text, "command-name")
	text = removeTaggedBlocks(text, "command-message")
	text = removeTaggedBlocks(text, "command-args")
	text = removeTaggedBlocks(text, "local-command-stdout")
	text = removeTaggedBlocks(text, "local-command-stderr")
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "<") && strings.Contains(text, ">") {
		return ""
	}
	return text
}

func removeTaggedBlocks(text, tag string) string {
	for {
		start := strings.Index(text, "<"+tag+">")
		if start < 0 {
			return text
		}
		endTag := "</" + tag + ">"
		end := strings.Index(text[start:], endTag)
		if end < 0 {
			return strings.TrimSpace(text[:start])
		}
		end += start + len(endTag)
		text = text[:start] + text[end:]
	}
}

// CountTokensHandler 处理 /v1/messages/count_tokens 请求
func CountTokensHandler(envCfg *config.EnvConfig, cfgManager *config.ConfigManager, channelScheduler *scheduler.ChannelScheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		middleware.ProxyAuthMiddleware(envCfg)(c)
		if c.IsAborted() {
			return
		}

		// 使用统一的请求体读取函数，应用大小限制
		bodyBytes, err := common.ReadRequestBody(c, envCfg.MaxRequestBodySize)
		if err != nil {
			// ReadRequestBody 已经返回了错误响应
			return
		}
		c.Set("requestBodyBytes", bodyBytes)

		var req struct {
			Model    string      `json:"model"`
			System   interface{} `json:"system"`
			Messages interface{} `json:"messages"`
			Tools    interface{} `json:"tools"`
		}
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		inputTokens := utils.EstimateRequestTokens(bodyBytes)

		c.JSON(200, gin.H{
			"input_tokens": inputTokens,
		})

		if envCfg.EnableResponseLogs {
			log.Printf("[Messages-Token] CountTokens本地估算: model=%s, input_tokens=%d", req.Model, inputTokens)
		}
	}
}
