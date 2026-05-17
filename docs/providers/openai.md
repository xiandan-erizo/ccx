# OpenAI GPT 配置指南

## 获取 API Key

1. 访问 [OpenAI Platform](https://platform.openai.com/)
2. 登录账号
3. 进入「API Keys」页面
4. 点击「Create new secret key」，复制生成的密钥

## 在 CCX 中添加渠道

### 基本配置

| 字段 | 值 |
|------|-----|
| 名称 | `OpenAI`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://api.openai.com/v1` |
| API Keys | 你的 OpenAI API Key（`sk-...`） |

### 配置步骤

1. 进入 CCX 管理界面，选择 **Chat** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`OpenAI`
   - **服务类型**：选择 `OpenAI Chat`
   - **Base URL**：`https://api.openai.com/v1`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### 模型白名单（可选）

```
gpt-4o
gpt-4o-mini
gpt-4-turbo
o1
o1-mini
o3
o3-mini
o4-mini
```

### 使用代理（可选）

如果你的服务器无法直接访问 OpenAI API，可以配置代理：

- **代理地址**：填写 HTTP/HTTPS/SOCKS5 代理地址，如 `http://127.0.0.1:7890`

或者使用第三方中转地址作为 Base URL。

## 可用模型

| 模型 | 说明 |
|------|------|
| `gpt-4o` | GPT-4o 多模态旗舰模型 |
| `gpt-4o-mini` | 轻量高性价比版本 |
| `o1` | 推理模型 |
| `o3` | 最新推理模型 |
| `o3-mini` | 轻量推理模型 |
| `o4-mini` | 最新轻量推理模型 |

## Images 入口配置

如果需要使用 OpenAI 的图片生成能力：

1. 选择 **Images** 入口
2. 添加渠道，配置与上面相同的 Base URL 和 API Key
3. 服务类型选择 `openai`

支持的图片模型：`dall-e-3`、`dall-e-2`、`gpt-image-1`

## Responses 入口配置

如果需要使用 Codex/Responses 协议：

1. 选择 **Responses** 入口
2. 添加渠道，配置与上面相同的 Base URL 和 API Key
3. 服务类型选择 `responses`

## 注意事项

- OpenAI API Key 以 `sk-` 开头
- 如果使用 Azure OpenAI，Base URL 和认证方式不同，请参考 Azure 文档
- 推理模型（o1/o3 系列）不支持 system message，CCX 会自动处理兼容性
