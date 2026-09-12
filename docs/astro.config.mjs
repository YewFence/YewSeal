// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://yewfence.github.io',
  base: '/YewSeal/',
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
          ],
        },
        {
          label: 'Integrations',
          items: [
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
