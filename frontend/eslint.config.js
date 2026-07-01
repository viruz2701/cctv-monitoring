// For more info, see https://github.com/storybookjs/eslint-plugin-storybook#configuration-flat-config-format
import storybook from "eslint-plugin-storybook";

import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import reactPlugin from 'eslint-plugin-react'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([globalIgnores(['dist']), {
  files: ['**/*.{ts,tsx}'],
  plugins: {
    react: reactPlugin,
  },
  extends: [
    js.configs.recommended,
    tseslint.configs.recommended,
    reactHooks.configs.flat.recommended,
    reactRefresh.configs.vite,
  ],
  languageOptions: {
    ecmaVersion: 2020,
    globals: globals.browser,
    parserOptions: {
      ecmaFeatures: {
        jsx: true,
      },
    },
  },
  rules: {
    'react-hooks/exhaustive-deps': ['warn', { additionalHooks: '(useMutation|useQuery)' }],
    'react/jsx-key': 'error',
  },
}, {
  files: ['**/*.stories.@(ts|tsx)'],
  plugins: {
    storybook,
  },
  rules: {
    'storybook/use-storybook-expect': 'error',
    'storybook/story-exports': 'error',
    'storybook/no-redundant-story-name': 'warn',
    'storybook/default-exports': 'error',
  },
}])
