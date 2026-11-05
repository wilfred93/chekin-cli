// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLinksValidator from 'starlight-links-validator';
import starlightLlmsTxt from 'starlight-llms-txt';

// Served from GitHub Pages at https://wilfred93.github.io/chekin-cli/.
export default defineConfig({
  site: 'https://wilfred93.github.io',
  base: '/chekin-cli',
  integrations: [
    starlight({
      title: 'chekin-cli',
      description: 'Unofficial command line tool for Chekin, built for scripts and AI agents.',
      social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/wilfred93/chekin-cli' }],
      editLink: { baseUrl: 'https://github.com/wilfred93/chekin-cli/edit/master/docs/' },
      lastUpdated: true,
      plugins: [starlightLinksValidator(), starlightLlmsTxt()],
      sidebar: [
        {
          label: 'Getting started',
          items: [
            { label: 'Home', slug: 'index' },
            { slug: 'install' },
            { slug: 'log-in' },
          ],
        },
        {
          label: 'Using it',
          items: [
            { slug: 'bookings' },
            { slug: 'booking-details' },
            { slug: 'output' },
            { slug: 'agents' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { slug: 'reference/commands' },
            { slug: 'troubleshooting' },
            { slug: 'how-it-works' },
          ],
        },
        {
          label: 'Project',
          items: [
            { slug: 'faq' },
            { slug: 'contributing' },
          ],
        },
      ],
    }),
  ],
});
