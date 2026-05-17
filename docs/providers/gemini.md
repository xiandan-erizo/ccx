# Gemini 配置指南

## 获取 API Key

1. 访问 [Google AI Studio](https://aistudio.google.com/)
2. 登录 Google 账号
3. 点击「Get API Key」
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### Gemini 入口（原生协议）

| 字段 | 值 |
|------|-----|
| 名称 | `Gemini`（自定义） |
| 服务类型 | `gemini` |
| Base URL | `https://generativelanguage.googleapis.com` |
| API Keys | 你的 Google AI API Key |

### 配置步骤（Gemini 入口）

1. 进入 CCX 管理界面，选择 **Gemini** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`Gemini`
   - **服务类型**：选择 `Gemini`
   - **Base URL**：`https://generativelanguage.googleapis.com`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### Chat 入口（OpenAI 兼容）

Gemini 也可以通过 OpenAI 兼容接口访问：

| 字段 | 值 |
|------|-----|
| 名称 | `Gemini (OpenAI)`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://generativelanguage.googleapis.com/v1beta/openai` |
| API Keys | 你的 Google AI API Key |

### 模型白名单（可选）

```
gemini-2.5-pro
gemini-2.5-flash
gemini-2.0-flash
gemini-1.5-pro
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `gemini-2.5-pro` | 最新旗舰模型，支持思考 |
| `gemini-2.5-flash` | 快速版本，支持思考 |
| `gemini-2.0-flash` | 高速多模态模型 |
| `gemini-1.5-pro` | 超长上下文（2M tokens） |

## 高级配置

### thought_signature 处理

部分第三方 Gemini 代理要求请求中包含 `thought_signature` 字段：

- **注入 dummy thought_signature**：启用 `injectDummyThoughtSignature` 选项
- **移除 thought_signature**：启用 `stripThoughtSignature` 选项（兼容旧版 API）

## 注意事项

- Gemini 原生入口使用 Google 的 generateContent 协议
- 如需通过 OpenAI Chat 协议访问 Gemini，使用 Chat 入口 + `openai` 服务类型 + OpenAI 兼容 Base URL
- 如果服务器在中国大陆，需要配置代理才能访问 Google API
- Gemini 的 Vision 能力通过原生协议自动支持，无需额外配置
