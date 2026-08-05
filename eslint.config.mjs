import { defineConfig } from 'eslint/config';
import baseConfig from './.config/eslint.config.mjs';

export default defineConfig([
  {
    ignores: [
      '**/*.log',
      '**/node_modules/',
      '**/dist/',
      '**/coverage/',
      '**/ci/',
      '**/.idea',
      '**/.eslintcache',
    ],
  },
  ...baseConfig,
]);
