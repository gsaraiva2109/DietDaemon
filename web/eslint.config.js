import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
    ],
    plugins: {
      'react-refresh': reactRefresh,
    },
    rules: {
      'react-refresh/only-export-components': [
        'warn',
        {
          allowConstantExport: true,
          allowExportNames: [
            'useAuth',
            'useDemo',
            'useTheme',
            'demoAvailable',
            'demoRange',
            'demoToday',
            'demoFoodSearch',
            'demoWeightTrend',
            'demoBodySummary',
            'sourceLabel',
            'DEMO_TARGETS',
            'DEMO_CONSUMED',
            'DEMO_MEALS',
            'DEMO_FOODS',
            'DEMO_TEMPLATES',
            'DEMO_WEIGHT',
            'DEMO_MEASUREMENTS',
            'DEMO_PROFILE',
          ],
        },
      ],
    },
    languageOptions: {
      globals: globals.browser,
    },
  },
])
