package responses

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/BenedictKing/ccx/internal/config"
	"github.com/BenedictKing/ccx/internal/converters"
	"github.com/BenedictKing/ccx/internal/handlers/common"
	"github.com/BenedictKing/ccx/internal/providers"
	"github.com/BenedictKing/ccx/internal/session"
	"github.com/BenedictKing/ccx/internal/types"
	"github.com/BenedictKing/ccx/internal/utils"
	"github.com/gin-gonic/gin"
)

const localCompactMaxTranscriptRunes = 240000
const localCompactMaxArgRunes = 8000
const localCompactMaxReasoningRunes = 12000
const localCompactDefaultMaxTokens = 8192

// PLACEHOLDER_COMPACT_LOCAL_CONTINUED

const localCompactSystemPrompt = `You are a conversation compressor. Create a concise handover document that captures the essential context of the conversation below.

Include:
1. Task Context: What the user is working on (project, files, goals)
2. Key Decisions: Important decisions made during the conversation
3. Current State: What's done, what's pending
4. Important Details: File paths, code patterns, configuration values, error messages needed to continue
5. Next Steps: What was about to happen or was requested

Rules:
- Preserve EXACT technical terms, file paths, function names, code snippets
- Include FULL context: paths, versions, configurations
- Omit verbose tool output details unless they contain critical information
- Be concise but preserve all information needed to continue the conversation
- Use markdown code blocks for code snippets with language tags
- NO assumptions, NO vague summaries - only document what was explicitly discussed`

func needsLocalCompact(upstream *config.UpstreamConfig) bool {
	return upstream.ServiceType != "responses"
}

func isNativeCompactUnsupported(status int) bool {
	return status == 404 || status == 405 || status == 501
}

// PLACEHOLDER_FORMAT_TRANSCRIPT

func formatItemsAsTranscript(items []types.ResponsesItem) string {
	var parts []string
	for _, item := range items {
		formatted := formatSingleItem(item)
		if formatted != "" {
			parts = append(parts, formatted)
		}
	}
	transcript := strings.Join(parts, "\n\n---\n\n")
	return truncateTranscript(transcript)
}

func formatSingleItem(item types.ResponsesItem) string {
	switch item.Type {
	case "message":
		return formatMessageItem(item)
	case "function_call":
		args := truncateRunes(item.Arguments, localCompactMaxArgRunes)
		if args == "" {
			args = truncateRunes(fmt.Sprintf("%v", item.Input), localCompactMaxArgRunes)
		}
		return fmt.Sprintf("  -> Tool Call: %s(%s)", item.Name, args)
	case "function_call_output":
		return "" // skip tool results
	case "reasoning":
		text := extractContentText(item.Content)
		if text == "" {
			if s, ok := item.Summary.(string); ok {
				text = s
			}
		}
		return "[Reasoning]\n" + truncateRunes(text, localCompactMaxReasoningRunes)
	default:
		text := extractContentText(item.Content)
		if text != "" {
			return fmt.Sprintf("[%s]\n%s", item.Type, truncateRunes(text, localCompactMaxReasoningRunes))
		}
		return ""
	}
}

func formatMessageItem(item types.ResponsesItem) string {
	label := "[User]"
	if item.Role == "assistant" {
		label = "[Assistant]"
	}
	text := extractContentText(item.Content)
	if text == "" {
		return ""
	}
	return label + "\n" + text
}

func extractContentText(content interface{}) string {
	if content == nil {
		return ""
	}
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var texts []string
		for _, block := range v {
			m, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			blockType, _ := m["type"].(string)
			switch blockType {
			case "input_text", "output_text", "text":
				if t, ok := m["text"].(string); ok && t != "" {
					texts = append(texts, t)
				}
			case "input_image":
				url, _ := m["image_url"].(string)
				if url == "" {
					if urlObj, ok := m["image_url"].(map[string]interface{}); ok {
						url, _ = urlObj["url"].(string)
					}
				}
				texts = append(texts, fmt.Sprintf("[Image: %s]", url))
			default:
				if t, ok := m["text"].(string); ok && t != "" {
					texts = append(texts, t)
				}
			}
		}
		return strings.Join(texts, "\n")
	case []types.ContentBlock:
		var texts []string
		for _, block := range v {
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

func truncateTranscript(transcript string) string {
	runes := []rune(transcript)
	if len(runes) <= localCompactMaxTranscriptRunes {
		return transcript
	}
	headSize := localCompactMaxTranscriptRunes * 20 / 100
	tailSize := localCompactMaxTranscriptRunes * 75 / 100
	omitted := len(runes) - headSize - tailSize
	log.Printf("[Compact-Local] transcript 截断: before=%d after=%d", len(runes), headSize+tailSize)
	return string(runes[:headSize]) +
		fmt.Sprintf("\n\n[... omitted %d characters during local compact ...]\n\n", omitted) +
		string(runes[len(runes)-tailSize:])
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// PLACEHOLDER_BUILD_REQUEST

func buildLocalCompactRequestBody(originalBody []byte, stream bool, sessionManager *session.SessionManager) ([]byte, error) {
	var originalReq types.ResponsesRequest
	if err := json.Unmarshal(originalBody, &originalReq); err != nil {
		return nil, fmt.Errorf("解析 compact 请求失败: %w", err)
	}

	// 收集上下文 items
	var allItems []types.ResponsesItem

	// 从 session 获取历史
	if originalReq.PreviousResponseID != "" && sessionManager != nil {
		sess, err := sessionManager.GetSessionByResponseID(originalReq.PreviousResponseID)
		if err == nil && len(sess.Messages) > 0 {
			allItems = append(allItems, sess.Messages...)
		} else {
			log.Printf("[Compact-Local] previous_response_id 未命中本地 session: %s", originalReq.PreviousResponseID)
		}
	}

	// 追加当前 input
	inputItems, _ := types.ParseResponsesInput(originalReq.Input)
	allItems = append(allItems, inputItems...)

	if len(allItems) == 0 {
		if originalReq.PreviousResponseID != "" {
			return nil, fmt.Errorf("无法本地 compact: previous_response_id 未命中且 input 为空")
		}
		return nil, fmt.Errorf("无法本地 compact: input 为空")
	}

	transcript := formatItemsAsTranscript(allItems)

	maxTokens := localCompactDefaultMaxTokens
	if originalReq.MaxTokens > 0 && originalReq.MaxTokens < maxTokens {
		maxTokens = originalReq.MaxTokens
	}

	compactReq := map[string]interface{}{
		"model":             originalReq.Model,
		"instructions":      localCompactSystemPrompt,
		"input":             []interface{}{map[string]interface{}{"type": "message", "role": "user", "content": []interface{}{map[string]interface{}{"type": "input_text", "text": transcript}}}},
		"stream":            stream,
		"max_output_tokens": maxTokens,
	}

	if originalReq.Temperature > 0 {
		compactReq["temperature"] = originalReq.Temperature
	}
	if originalReq.TopP > 0 {
		compactReq["top_p"] = originalReq.TopP
	}
	if originalReq.User != "" {
		compactReq["user"] = originalReq.User
	}

	return utils.MarshalJSONNoEscape(compactReq)
}

// PLACEHOLDER_TRY_LOCAL_COMPACT

func tryLocalCompactWithKey(
	c *gin.Context,
	upstream *config.UpstreamConfig,
	apiKey string,
	bodyBytes []byte,
	envCfg *config.EnvConfig,
	cfgManager *config.ConfigManager,
	sessionManager *session.SessionManager,
) (bool, *compactError) {
	// 解析原始请求判断 stream
	var originalReq types.ResponsesRequest
	if err := json.Unmarshal(bodyBytes, &originalReq); err != nil {
		return false, &compactError{status: 400, body: []byte(`{"error":"解析 compact 请求失败"}`), shouldFailover: false, err: err}
	}

	stream := originalReq.Stream
	log.Printf("[Compact-Local] 使用本地 compact: serviceType=%s model=%s stream=%v", upstream.ServiceType, originalReq.Model, stream)

	// 构建本地 compact 请求体
	localBody, err := buildLocalCompactRequestBody(bodyBytes, stream, sessionManager)
	if err != nil {
		return false, &compactError{status: 400, body: []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())), shouldFailover: false, err: err}
	}

	// 通过 provider 转换并构建上游请求
	provider := &providers.ResponsesProvider{SessionManager: sessionManager}
	req, _, err := provider.ConvertBodyToProviderRequest(c, upstream, apiKey, localBody, "/v1/responses")
	if err != nil {
		return false, &compactError{status: 500, body: []byte(`{"error":"构建本地 compact 上游请求失败"}`), shouldFailover: true, err: err}
	}

	// 发送请求
	resp, err := common.SendRequest(req, upstream, envCfg, stream, "Responses")
	if err != nil {
		return false, &compactError{status: 502, body: []byte(`{"error":"本地 compact 上游请求失败"}`), shouldFailover: true, err: err}
	}
	defer resp.Body.Close()

	// 错误处理
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		respBody = utils.DecompressGzipIfNeeded(resp, respBody)
		shouldFailover, _ := common.ShouldRetryWithNextKey(resp.StatusCode, respBody, cfgManager.GetFuzzyModeEnabled(), "Responses")
		return false, &compactError{status: resp.StatusCode, body: respBody, shouldFailover: shouldFailover}
	}

	if stream {
		return handleLocalCompactStream(c, resp, upstream.ServiceType, originalReq, sessionManager)
	}
	return handleLocalCompactNonStream(c, resp, upstream.ServiceType, originalReq, sessionManager)
}

// PLACEHOLDER_HANDLE_NONSTREAM

func handleLocalCompactNonStream(
	c *gin.Context,
	resp *http.Response,
	upstreamType string,
	originalReq types.ResponsesRequest,
	sessionManager *session.SessionManager,
) (bool, *compactError) {
	respBody, _ := io.ReadAll(resp.Body)
	respBody = utils.DecompressGzipIfNeeded(resp, respBody)

	// 转换响应
	converter := converters.NewConverter(upstreamType)
	respMap, err := converters.JSONToMap(respBody)
	if err != nil {
		return false, &compactError{status: 502, body: respBody, shouldFailover: true, err: err}
	}

	responsesResp, err := converter.FromProviderResponse(respMap, "")
	if err != nil {
		return false, &compactError{status: 502, body: respBody, shouldFailover: true, err: err}
	}

	// 补齐字段
	if responsesResp.ID == "" {
		responsesResp.ID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	if responsesResp.Status == "" {
		responsesResp.Status = "completed"
	}
	if responsesResp.Model == "" {
		responsesResp.Model = originalReq.Model
	}
	if originalReq.PreviousResponseID != "" {
		responsesResp.PreviousID = originalReq.PreviousResponseID
	}

	// 写回 session
	writeCompactedSession(responsesResp, originalReq, sessionManager)

	utils.ForwardResponseHeaders(resp.Header, c.Writer)
	c.JSON(200, responsesResp)
	return true, nil
}

// PLACEHOLDER_HANDLE_STREAM

func handleLocalCompactStream(
	c *gin.Context,
	resp *http.Response,
	upstreamType string,
	originalReq types.ResponsesRequest,
	sessionManager *session.SessionManager,
) (bool, *compactError) {
	needConvert := upstreamType != "responses"
	var converterState *any
	var summaryBuf strings.Builder
	var responseID string

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)
	flusher, _ := c.Writer.(http.Flusher)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var eventsToSend []string

		if needConvert {
			switch upstreamType {
			case "claude":
				eventsToSend = converters.ConvertClaudeMessagesToResponses(
					c.Request.Context(), originalReq.Model, nil, nil, []byte(line), converterState,
				)
			case "gemini":
				eventsToSend = converters.ConvertGeminiStreamToResponses(
					c.Request.Context(), originalReq.Model, nil, nil, []byte(line), converterState,
				)
			default:
				eventsToSend = converters.ConvertOpenAIChatToResponses(
					c.Request.Context(), originalReq.Model, nil, nil, []byte(line), converterState,
				)
			}
		} else {
			eventsToSend = []string{line + "\n"}
		}

		for _, event := range eventsToSend {
			// 收集摘要文本
			collectStreamSummary(event, &summaryBuf, &responseID)

			if _, err := c.Writer.Write([]byte(event)); err != nil {
				return true, nil
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}

	// 写回 session
	if summaryText := summaryBuf.String(); summaryText != "" {
		if responseID == "" {
			responseID = fmt.Sprintf("resp_%d", time.Now().UnixNano())
		}
		resp := &types.ResponsesResponse{
			ID:     responseID,
			Model:  originalReq.Model,
			Status: "completed",
			Output: []types.ResponsesItem{{
				Type:    "message",
				Role:    "assistant",
				Content: []types.ContentBlock{{Type: "output_text", Text: summaryText}},
			}},
		}
		if originalReq.PreviousResponseID != "" {
			resp.PreviousID = originalReq.PreviousResponseID
		}
		writeCompactedSession(resp, originalReq, sessionManager)
	}

	return true, nil
}

func collectStreamSummary(event string, buf *strings.Builder, responseID *string) {
	if strings.Contains(event, `"response.output_text.delta"`) {
		// 提取 delta 文本
		var data map[string]interface{}
		if idx := strings.Index(event, "data: "); idx >= 0 {
			jsonStr := event[idx+6:]
			if json.Unmarshal([]byte(jsonStr), &data) == nil {
				if delta, ok := data["delta"].(string); ok {
					buf.WriteString(delta)
				}
			}
		}
	}
	if strings.Contains(event, `"response.completed"`) && *responseID == "" {
		var data map[string]interface{}
		if idx := strings.Index(event, "data: "); idx >= 0 {
			jsonStr := event[idx+6:]
			if json.Unmarshal([]byte(jsonStr), &data) == nil {
				if resp, ok := data["response"].(map[string]interface{}); ok {
					if id, ok := resp["id"].(string); ok && id != "" {
						*responseID = id
					}
				}
			}
		}
	}
}

// PLACEHOLDER_SESSION_WRITE

func writeCompactedSession(resp *types.ResponsesResponse, originalReq types.ResponsesRequest, sessionManager *session.SessionManager) {
	if sessionManager == nil {
		return
	}
	// 尊重 store 语义
	if originalReq.Store != nil && !*originalReq.Store {
		return
	}

	// 提取摘要文本
	summaryText := ""
	for _, item := range resp.Output {
		if item.Type == "message" && item.Role == "assistant" {
			summaryText = extractContentText(item.Content)
			break
		}
	}
	if summaryText == "" {
		return
	}

	messages := []types.ResponsesItem{
		{
			Type:    "message",
			Role:    "user",
			Content: "Conversation context has been compacted. Continue from this summary.",
		},
		{
			Type:    "message",
			Role:    "assistant",
			Content: summaryText,
		},
	}

	totalTokens := resp.Usage.InputTokens + resp.Usage.OutputTokens
	sessionManager.CreateCompactedSession(resp.ID, messages, totalTokens)
}
