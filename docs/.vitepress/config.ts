import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/YewSeal/',
  lang: 'zh-CN',
  title: 'YewSeal',
  description: '加密配置文件管理工具',

  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/getting-started' },
      { text: '命令参考', link: '/commands/init' },
      { text: 'GitHub', link: 'https://github.com/YewFence/YewSeal' }
    ],

    sidebar: [
      {
        text: '指南',
        items: [
          { text: '快速开始', link: '/guide/getting-started' },
          { text: '配置说明', link: '/guide/configuration' },
          { text: 'Shell 补全', link: '/guide/completion' }
        ]
      },
      {
        text: '命令参考',
        items: [
          { text: 'init - 初始化项目', link: '/commands/init' },
          { text: 'encrypt - 加密文件', link: '/commands/encrypt' },
          { text: 'decrypt - 解密文件', link: '/commands/decrypt' },
          { text: 'edit - 编辑加密文件', link: '/commands/edit' },
          { text: 'view - 查看加密文件', link: '/commands/view' },
          { text: 'diff - 比较差异', link: '/commands/diff' },
          { text: 'sync - 同步密钥', link: '/commands/sync' },
          { text: 'check - 检查依赖', link: '/commands/check' }
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
