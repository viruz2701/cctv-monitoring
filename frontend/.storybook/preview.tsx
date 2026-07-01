import type { Preview } from '@storybook/react-vite'

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },

    // P2-MED-18: Storybook backgrounds (light/dark)
    backgrounds: {
      default: 'light',
      values: [
        { name: 'Light', value: '#ffffff' },
        { name: 'Dark', value: '#0f172a' },
        { name: 'Slate 50', value: '#f8fafc' },
        { name: 'Slate 900', value: '#0f172a' },
      ],
    },

    // P2-MED-18: Viewport sizes (mobile 375, tablet 768, desktop 1280)
    viewport: {
      defaultViewport: 'responsive',
      viewports: {
        mobile: {
          name: 'Mobile 375',
          styles: { width: '375px', height: '812px' },
        },
        tablet: {
          name: 'Tablet 768',
          styles: { width: '768px', height: '1024px' },
        },
        desktop: {
          name: 'Desktop 1280',
          styles: { width: '1280px', height: '800px' },
        },
        desktopHD: {
          name: 'Desktop 1920',
          styles: { width: '1920px', height: '1080px' },
        },
      },
    },

    a11y: {
      // 'todo' - show a11y violations in the test UI only
      // 'error' - fail CI on a11y violations
      // 'off' - skip a11y checks entirely
      test: 'todo',
      config: {
        rules: [
          { id: 'color-contrast', enabled: true },
          { id: 'aria-valid-attr', enabled: true },
          { id: 'aria-valid-attr-value', enabled: true },
          { id: 'button-name', enabled: true },
          { id: 'image-alt', enabled: true },
          { id: 'label', enabled: true },
          { id: 'link-name', enabled: true },
        ],
      },
    },
  },
}

export default preview
