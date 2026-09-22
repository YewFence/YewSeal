// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { satteri } from '@astrojs/markdown-satteri';

// GitHub Pages serves this site under /YewSeal/, but Astro does not rebase
// root-absolute links authored in Markdown with the configured base: a
// hand-written [Tutorial](/guide/tutorial) would render as href="/guide/tutorial"
// and 404 in production. This Sätteri HAST plugin prefixes every site-internal
// anchor href with the base at build time, so keep authoring /guide/... links.
const BASE = '/YewSeal/';

const rebaseInternalLinks = {
  name: 'yewseal-rebase-internal-links',
  element: {
    filter: ['a'],
    visit(node, ctx) {
      const href = node.properties?.href;
      if (typeof href === 'string' && href.startsWith('/') && !href.startsWith('//')) {
        ctx.setProperty(node, 'href', BASE.replace(/\/+$/, '') + href);
      }
    },
  },
}

export default defineConfig({
  site: 'https://yewfence.github.io',
  base: BASE,
  markdown: {
    processor: satteri({ hastPlugins: [rebaseInternalLinks] }),
  },
  integrations: [
    starlight({
      title: 'YewSeal',
      description: 'Encrypted configuration file manager',
      defaultLocale: 'en',
      social: [
        { label: 'GitHub', icon: 'github', href: 'https://github.com/YewFence/YewSeal' },
      ],
      sidebar: [
        {
          label: 'Get started',
          items: [
            { label: 'Installation', link: '/guide/installation/' },
            { label: 'Tutorial', link: '/guide/tutorial/' },
            { label: 'Working with a team', link: '/guide/working-with-a-team/' },
          ],
        },
        {
          label: 'Core guides',
          items: [
            { label: 'Configuration', link: '/guide/configuration/' },
            { label: 'Target selection', link: '/guide/target-selection/' },
            { label: 'Decryption results and strict mode', link: '/guide/decryption-results/' },
            { label: 'Cleaning local plaintext', link: '/guide/plaintext-cleanup/' },
            { label: 'Glossary', link: '/guide/glossary/' },
          ],
        },
        {
          label: 'Integrations',
          items: [
            { label: 'Agent skills', link: '/guide/agent-skills/' },
            { label: 'Interop with SOPS', link: '/guide/sops/' },
            { label: 'External private key sources', link: '/guide/private-keys/' },
            { label: 'Running with Docker', link: '/guide/docker/' },
            { label: 'CI/CD integration', link: '/guide/ci-cd/' },
          ],
        },
        {
          label: 'CLI reference',
          items: [{ autogenerate: { directory: 'references' } }],
        },
      ],
    }),
  ],
});
