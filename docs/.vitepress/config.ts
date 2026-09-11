import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/YewSeal/',
  lang: 'en-US',
  title: 'YewSeal',
  description: 'Encrypted configuration file manager',

  themeConfig: {
    nav: [
      { text: 'Guides', link: '/guide/getting-started' },
      { text: 'CLI reference', link: '/references/yews' },
      { text: 'GitHub', link: 'https://github.com/YewFence/YewSeal' }
    ],

    sidebar: [
      {
        text: 'Guides',
        items: [
          { text: 'Getting started', link: '/guide/getting-started' },
          { text: 'Configuration', link: '/guide/configuration' },
          { text: 'Target selection', link: '/guide/target-selection' },
          { text: 'Workflows', link: '/guide/workflows' },
          { text: 'Decryption results and strict mode', link: '/guide/decryption-results' },
          { text: 'Interop with SOPS', link: '/guide/sops' },
          { text: 'External private key sources', link: '/guide/private-keys' },
          { text: 'Running with Docker', link: '/guide/docker' },
          { text: 'CI/CD integration', link: '/guide/ci-cd' },
          { text: 'Shell completion', link: '/guide/completion' }
        ]
      },
      {
        text: 'CLI reference (generated)',
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
    }
  }
})
