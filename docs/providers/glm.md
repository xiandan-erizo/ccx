# 智谱 GLM 配置指南

## 获取 API Key

1. 访问 [智谱 AI 开放平台](https://open.bigmodel.cn/)（品牌升级为 [Z.AI](https://z.ai)）
2. 注册并登录账号
3. 进入「API Keys」管理页面
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### 方式一：Chat 入口（OpenAI 兼容协议）

| 字段 | 值 |
|------|-----|
| 名称 | `智谱 GLM`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://open.bigmodel.cn/api/paas/v4` |
| API Keys | 你的智谱 API Key |

#### 配置步骤

1. 进入 CCX 管理界面，选择 **Chat** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`智谱 GLM`
   - **服务类型**：选择 `OpenAI Chat`
   - **Base URL**：`https://open.bigmodel.cn/api/paas/v4`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### 方式二：Messages 入口（Anthropic 兼容协议）

适用于 Claude Code CLI 等使用 Claude Messages 协议的工具。

| 字段 | 值 |
|------|-----|
| 名称 | `智谱 GLM Claude`（自定义） |
| 服务类型 | `claude` |
| Base URL | `https://open.bigmodel.cn/api/anthropic` |
| API Keys | 你的智谱 API Key |

#### 模型映射（Messages 入口推荐）

| 请求模型 | 重定向到 |
|----------|----------|
| `opus` | `glm-5.1` |
| `sonnet` | `glm-5` |
| `haiku` | `glm-5-turbo` |

### 模型白名单（可选）

```
glm-5.1
glm-5
glm-5-turbo
glm-4.7
glm-4.6
glm-4.5
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `glm-5.1` | 最新旗舰，面向 Agentic 工程，744B 参数 |
| `glm-5` | 上一代旗舰 |
| `glm-5-turbo` | 快速版本 |
| `glm-4.7` | MoE 架构 |
| `glm-4.6` | 多模态版本（支持视觉） |
| `glm-4.5` | MoE 架构，106B 总参 / 12B 激活 |

## 注意事项

- 智谱 API 同时兼容 OpenAI Chat 和 Anthropic Messages 协议
- OpenAI 兼容 Base URL 使用 `/v4` 路径，不要加 `/chat/completions`
- Anthropic 兼容 Base URL 为 `https://open.bigmodel.cn/api/anthropic`
- 智谱的 API Key 格式为 `xxxxxxxx.yyyyyyyy`，整个字符串都是 Key
- 智谱已品牌升级为 Z.AI，但 `open.bigmodel.cn` 端点仍然有效
