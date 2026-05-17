# 小米 MiMo 配置指南

## 获取 API Key

MiMo 模型可通过以下平台访问：
- [硅基流动 SiliconFlow](https://cloud.siliconflow.cn/)（推荐）
- [小米 MiMo 官网](https://mimo.xiaomi.com/)

### 通过硅基流动获取

1. 访问 [硅基流动](https://cloud.siliconflow.cn/)
2. 注册并登录账号
3. 进入「API Keys」页面
4. 创建新的 API Key 并复制

## 在 CCX 中添加渠道

### 通过硅基流动访问

| 字段 | 值 |
|------|-----|
| 名称 | `MiMo (SiliconFlow)`（自定义） |
| 服务类型 | `openai` |
| Base URL | `https://api.siliconflow.cn/v1` |
| API Keys | 你的 SiliconFlow API Key |

#### 配置步骤

1. 进入 CCX 管理界面，选择 **Chat** 入口
2. 点击「添加渠道」
3. 填写以下信息：
   - **名称**：`MiMo`
   - **服务类型**：选择 `OpenAI Chat`
   - **Base URL**：`https://api.siliconflow.cn/v1`
   - **API Keys**：粘贴你的 API Key
4. 点击保存

### 模型白名单（可选）

```
XiaomiMiMo/MiMo-V2.5-Pro
XiaomiMiMo/MiMo-V2.5
XiaomiMiMo/MiMo-V2-Flash
```

### 模型映射（可选）

```json
{
  "mimo-pro": "XiaomiMiMo/MiMo-V2.5-Pro",
  "mimo": "XiaomiMiMo/MiMo-V2.5",
  "mimo-flash": "XiaomiMiMo/MiMo-V2-Flash"
}
```

## 可用模型

| 模型 | 说明 |
|------|------|
| `XiaomiMiMo/MiMo-V2.5-Pro` | 最新旗舰，1.02T 总参 / 42B 激活 |
| `XiaomiMiMo/MiMo-V2.5` | 310B 总参 / 15B 激活，原生多模态 |
| `XiaomiMiMo/MiMo-V2-Flash` | 309B 总参 / 15B 激活，高速推理 |

::: tip
硅基流动上的模型 ID 格式为 `组织名/模型名`，如 `XiaomiMiMo/MiMo-V2.5-Pro`。使用时需要填写完整标识。
:::

## 注意事项

- MiMo 通过兼容 OpenAI 协议的平台访问
- 硅基流动国内 Base URL：`https://api.siliconflow.cn/v1`
- 硅基流动国际 Base URL：`https://api.siliconflow.com/v1`
- MiMo 是推理模型，支持 `reasoning_content` 字段返回思考过程
- 硅基流动也提供 Anthropic 兼容端点（`/anthropic/v1/messages`）
