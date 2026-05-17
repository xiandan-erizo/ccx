import type { DefaultTheme, LocaleSpecificConfig } from 'vitepress'

export const enConfig: LocaleSpecificConfig<DefaultTheme.Config> = {
  title: 'CCX Docs',
  description: 'AI API Proxy & Protocol Translation Gateway',
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/en/guide/getting-started' },
      { text: 'Provider Setup', link: '/en/providers/' },
    ],
    sidebar: {
      '/en/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Quick Start', link: '/en/guide/getting-started' },
            { text: 'Deployment', link: '/en/guide/deployment' },
            { text: 'Environment Variables', link: '/en/guide/environment' },
          ],
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Architecture', link: '/en/guide/architecture' },
            { text: 'Development', link: '/en/guide/development' },
            { text: 'Release', link: '/en/guide/release' },
            { text: 'Contributing', link: '/en/guide/contributing' },
          ],
        },
      ],
      '/en/providers/': [
        {
          text: 'Provider Setup',
          items: [
            { text: 'Overview', link: '/en/providers/' },
            { text: 'DeepSeek', link: '/en/providers/deepseek' },
            { text: 'GLM (Zhipu AI)', link: '/en/providers/glm' },
            { text: 'MiniMax', link: '/en/providers/minimax' },
            { text: 'Kimi (Moonshot)', link: '/en/providers/kimi' },
            { text: 'OpenAI GPT', link: '/en/providers/openai' },
            { text: 'Xiaomi MiMo', link: '/en/providers/mimo' },
            { text: 'Claude', link: '/en/providers/claude' },
            { text: 'Gemini', link: '/en/providers/gemini' },
          ],
        },
      ],
    },
  },
}
