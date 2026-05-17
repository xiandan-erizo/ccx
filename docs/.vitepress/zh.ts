import type { DefaultTheme, LocaleSpecificConfig } from 'vitepress'

export const zhConfig: LocaleSpecificConfig<DefaultTheme.Config> = {
  title: 'CCX 文档',
  description: 'AI API 代理与协议转换网关',
  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/getting-started' },
      { text: '配置教程', link: '/providers/' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: '入门',
          items: [
            { text: '快速开始', link: '/guide/getting-started' },
            { text: '部署', link: '/guide/deployment' },
            { text: '环境变量', link: '/guide/environment' },
          ],
        },
        {
          text: '深入',
          items: [
            { text: '架构说明', link: '/guide/architecture' },
            { text: '开发指南', link: '/guide/development' },
            { text: '发布流程', link: '/guide/release' },
            { text: '贡献指南', link: '/guide/contributing' },
          ],
        },
      ],
      '/providers/': [
        {
          text: '配置教程',
          items: [
            { text: '概览', link: '/providers/' },
            { text: 'DeepSeek', link: '/providers/deepseek' },
            { text: '智谱 GLM', link: '/providers/glm' },
            { text: 'MiniMax', link: '/providers/minimax' },
            { text: 'Kimi (月之暗面)', link: '/providers/kimi' },
            { text: 'OpenAI GPT', link: '/providers/openai' },
            { text: '小米 MiMo', link: '/providers/mimo' },
            { text: 'Claude', link: '/providers/claude' },
            { text: 'Gemini', link: '/providers/gemini' },
          ],
        },
      ],
    },
    outline: { label: '页面导航' },
    lastUpdated: { text: '最后更新于' },
    docFooter: { prev: '上一页', next: '下一页' },
  },
}
