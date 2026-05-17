import { defineConfig } from 'vitepress'
import { zhConfig } from './zh'
import { enConfig } from './en'

export default defineConfig({
  title: 'CCX',
  description: 'AI API Proxy & Protocol Translation Gateway',
  base: '/ccx/',
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/ccx/logo.svg' }],
  ],
  locales: {
    root: {
      label: '简体中文',
      lang: 'zh-CN',
      ...zhConfig,
    },
    en: {
      label: 'English',
      lang: 'en',
      ...enConfig,
    },
  },
  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/BenedictKing/ccx' },
    ],
    search: {
      provider: 'local',
    },
  },
})
