# MiniMax 配置指南

## 获取 API Key

1. 访问 [MiniMax 开放平台](https://platform.minimax.io/)
2. 注册并登录账号
3. 进入「接口密钥」页面
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### 方式一：Chat 入口（OpenAI 兼容协议）

| 字段 | 值 |
|------|-----|
| 名称 | `MiniMax`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://api.minimax.io/v1` |
| API Keys | 你的 MiniMax API Key |

#### 配置步骤

1. 进入 CCX 管理界面，选择 **Chat** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`MiniMax`
   - **服务类型**：选择 `OpenAI Chat`
   - **Base URL**：`https://api.minimax.io/v1`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### 方式二：Messages 入口（Anthropic 兼容协议）

适用于 Claude Code CLI 等使用 Claude Messages 协议的工具。

| 字段 | 值 |
|------|-----|
| 名称 | `MiniMax Claude`（自定义） |
| 服务类型 | `claude` |
| Base URL | `https://api.minimax.io/anthropic` |
| API Keys | 你的 MiniMax API Key |

#### 模型映射（Messages 入口推荐）

| 请求模型 | 重定向到 |
|----------|----------|
| `opus` | `MiniMax-M2.7` |
| `sonnet` | `MiniMax-M2.7` |
| `haiku` | `MiniMax-M2.7-highspeed` |

### 模型白名单（可选）

```
MiniMax-M2.7
MiniMax-M2.7-highspeed
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `MiniMax-M2.7` | 最新旗舰，230B 总参数 MoE，自进化能力 |
| `MiniMax-M2.7-highspeed` | 高速版本 |

## 注意事项

- MiniMax 同时兼容 OpenAI Chat 和 Anthropic Messages 协议
- 官方域名已更新为 `api.minimax.io`（旧域名 `api.minimaxi.com` 可能仍有效）
- OpenAI 兼容 Base URL 包含 `/v1`
- Anthropic 兼容 Base URL 为 `https://api.minimax.io/anthropic`
