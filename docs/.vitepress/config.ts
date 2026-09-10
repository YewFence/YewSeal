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
          { text: '外部私钥来源', link: '/guide/private-keys' },
          { text: 'Docker 运行', link: '/guide/docker' },
          { text: 'CI/CD 集成', link: '/guide/ci-cd' },
          { text: 'Shell 补全', link: '/guide/completion' }
        ]
      },
      {
        text: '命令参考',
        items: [
          { text: 'init - 初始化项目', link: '/commands/init' },
          { text: 'encrypt - 加密文件', link: '/commands/encrypt' },
          { text: 'decrypt - 解密文件', link: '/commands/decrypt' },
          { text: 'plan - 预览选择', link: '/commands/plan' },
          { text: 'edit - 编辑加密文件', link: '/commands/edit' },
          { text: 'view - 查看加密文件', link: '/commands/view' },
          { text: 'diff - 比较差异', link: '/commands/diff' }
        ]
      },
      {
        text: 'CLI 参考（自动生成）',
        collapsed: true,
        items: [
          { text: 'yews', link: '/references/yews' },
          { text: 'yews init', link: '/references/yews_init' },
          { text: 'yews encrypt', link: '/references/yews_encrypt' },
          { text: 'yews decrypt', link: '/references/yews_decrypt' },
          { text: 'yews plan', link: '/references/yews_plan' },
          { text: 'yews edit', link: '/references/yews_edit' },
          { text: 'yews view', link: '/references/yews_view' },
          { text: 'yews diff', link: '/references/yews_diff' },
          { text: 'yews completion', link: '/references/yews_completion' }
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
