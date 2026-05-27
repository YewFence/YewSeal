import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/YewSeal/',
  lang: 'zh-CN',
  title: 'YewSeal',
  description: '加密配置文件管理工具',

  themeConfig: {
    nav: [
      { text: 'Shell 补全', link: '/guide/completion' },
      { text: 'GitHub', link: 'https://github.com/YewFence/YewSeal' }
    ],

    sidebar: [
      {
        text: '指南',
        items: [
          { text: 'Shell 补全', link: '/guide/completion' }
        ]
      }
    ],

    search: {
      provider: 'local'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/YewFence/YewSeal' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © YewFence'
    },

    docFooter: {
      prev: '上一页',
      next: '下一页'
    },

    outline: {
      label: '本页目录'
    },

    lastUpdated: {
      text: '最后更新'
    }
  }
})
