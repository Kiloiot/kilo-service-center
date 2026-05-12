import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';
import simpleImportSort from 'eslint-plugin-simple-import-sort';
import importPlugin from 'eslint-plugin-import';

export default tseslint.config(
  // Global ignores
  {
    ignores: [
      'dist',
      'node_modules',
      'coverage',
      'vite.config.ts',
      'vitest.config.ts',
      'playwright.config.ts',
      'e2e/**',
      // Test setup files are excluded from KC-Web/tsconfig.app.json (they are
      // contributor-only via the kc-web-dev vitest harness). Excluding them
      // from ESLint here prevents @typescript-eslint/parser from failing on
      // "file not found in any of the provided project(s)".
      'src/test/**',
      // Generated gRPC-web stubs - auto-generated code
      'src/services/grpc/kilocenter_pb.js',
      'src/services/grpc/kilocenter_pb.d.ts',
      'src/services/grpc/kilocenter_pb_service.js',
      'src/services/grpc/kilocenter_pb_service.d.ts',
      'src/services/grpc/identity_pb.js',
      'src/services/grpc/identity_pb.d.ts',
      'src/services/grpc/identity_pb_service.js',
      'src/services/grpc/identity_pb_service.d.ts',
      'src/services/grpc/core_pb.js',
      'src/services/grpc/core_pb.d.ts',
      'src/services/grpc/core_pb_service.js',
      'src/services/grpc/core_pb_service.d.ts',
    ],
  },
  // Main configuration for all TypeScript files
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: './tsconfig.app.json',
      },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      'simple-import-sort': simpleImportSort,
      import: importPlugin,
    },
    settings: {
      'import/resolver': {
        typescript: {
          project: './tsconfig.app.json',
        },
      },
    },
    rules: {
      // React hooks rules
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],

      // MANDATORY: Import sorting with groups
      'simple-import-sort/imports': [
        'error',
        {
          groups: [
            // React and react-dom
            ['^react', '^react-dom'],
            // External packages
            ['^@?\\w'],
            // Internal aliases
            [
              '^@/',
              '^@app/',
              '^@modules/',
              '^@components/',
              '^@ui/',
              '^@layouts/',
              '^@services/',
              '^@contexts/',
              '^@hooks/',
              '^@utils/',
              '^@constants/',
              '^@styles/',
              '^@router/',
              '^@types/',
              '^@locales/',
              '^@assets/',
              '^@config/',
              '^@theme/',
            ],
            // Relative imports
            ['^\\.'],
            // Style imports
            ['^.+\\.css$'],
          ],
        },
      ],
      'simple-import-sort/exports': 'error',

      // MANDATORY: Block restricted imports (error, not warn)
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['../../*', '../../../*'],
              message: 'Deep relative imports banned. Use @aliases.',
            },
            {
              group: ['@mui/icons-material/*', '@mui/icons-material'],
              message: 'Import icons from @theme/icons only.',
            },
            // Block deep relative imports to constants - use @constants/* aliases
            {
              group: ['../constants/*', '../../constants/*', '../../../constants/*'],
              message: 'Use @constants/app or @constants/messages aliases for constants.',
            },
          ],
        },
      ],

      // MANDATORY: Force barrel exports for modules - enforce barrels for feature modules
      // NOTE: This rule is configured to allow relative imports within modules
      // External consumers of modules MUST use barrel exports (@modules/*/index)
      'import/no-internal-modules': [
        'error',
        {
          allow: [
            // React standard imports
            'react-dom/client',
            // Module barrels only - external consumers use index.ts
            '@modules/*/index',
            // Cross-module hook imports (avoids pulling in heavy page components via barrel)
            '@modules/*/hooks',
            // Theme registry
            '@theme/icons',
            '@theme/index',
            // MUI components and date pickers
            '@mui/material/**',
            '@mui/x-date-pickers/**',
            // Shared components (not feature modules)
            '@components/common/**',
            '@components/layout/**',
            // Config files
            '@config/**',
            // Local relative imports within modules (pages, components subdirs)
            '**/pages/**',
            '**/components/**',
            // gRPC-web generated stubs and google-protobuf
            '@services/grpc/**',
            'google-protobuf/**',
            // Leaflet CSS and asset imports required for map rendering
            'leaflet/**',
          ],
        },
      ],

      // MANDATORY: Ban hardcoded hex/RGB color literals
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Literal[value=/^#[0-9A-Fa-f]{3,8}$/]',
          message: 'Hex colors banned. Use theme.palette.* tokens.',
        },
        {
          selector: 'Literal[value=/^rgb\\(|^rgba\\(/]',
          message: 'RGB colors banned. Use theme.palette.* tokens.',
        },
      ],
    },
  },
  // MANDATORY: no-default-export for shared layers
  {
    files: [
      'src/contexts/**/*.{ts,tsx}',
      'src/context/**/*.{ts,tsx}',
      'src/hooks/**/*.{ts,tsx}',
      'src/utils/**/*.{ts,tsx}',
      'src/services/**/*.{ts,tsx}',
      'src/ui/**/*.{ts,tsx}',
      'src/router/**/*.{ts,tsx}',
      'src/config/**/*.{ts,tsx}',
      'src/constants/**/*.{ts,tsx}',
    ],
    ignores: [
      // Explicit exceptions for React Router lazy loading
      'src/modules/**/pages/**',
      'src/router/AppRouter.tsx',
    ],
    rules: {
      'import/no-default-export': 'error',
    },
  },
  // Exception: Theme definition file may use hex/RGB colors
  {
    files: ['**/src/theme/index.ts'],
    rules: {
      'no-restricted-syntax': 'off',
    },
  },
  // Exception: Icon registry may import from @mui/icons-material
  {
    files: ['**/src/theme/icons.ts'],
    rules: {
      'no-restricted-imports': 'off',
    },
  },
  // MANDATORY: Block inline UI literals, hex colors, import.meta.env, and console
  // Scoped to production code only - tests may use inline strings for assertions
  // NOTE: All selectors merged into single block to prevent override conflicts
  {
    files: ['src/**/*.{ts,tsx}'],
    ignores: [
      '**/*.test.{ts,tsx}',
      '**/__tests__/**',
      '**/test/**',
      'src/config/env.ts',
      'src/theme/**/*.ts',
      'src/utils/logger.ts',
    ],
    rules: {
      'no-console': 'error',
      'no-restricted-syntax': [
        'error',
        // Hex/RGB color bans - use theme.palette.* tokens
        {
          selector: 'Literal[value=/^#[0-9A-Fa-f]{3,8}$/]',
          message: 'Hex colors banned. Use theme.palette.* tokens.',
        },
        {
          selector: 'Literal[value=/^rgb\\(|^rgba\\(/]',
          message: 'RGB colors banned. Use theme.palette.* tokens.',
        },
        // Block string literals in JSX text content (4+ chars to allow punctuation)
        {
          selector: 'JSXText:matches([value=/[a-zA-Z]{4,}/])',
          message: 'UI text must come from @constants/messages or @constants/app.',
        },
        // Block string literals in common UI attribute props
        {
          selector:
            'JSXAttribute[name.name=/^(title|label|placeholder|helperText|message|alt|aria-label)$/] > Literal',
          message: 'UI attribute text must come from @constants/messages.',
        },
        // Block import.meta.env access - must use @config/env
        {
          selector:
            'MemberExpression[object.object.name="import"][object.property.name="meta"][property.name="env"]',
          message: 'import.meta.env access banned. Import from @config/env.',
        },
      ],
    },
  },
  // Exception: Blueprints module allows underscore-prefixed unused vars (React Query destructuring pattern)
  {
    files: ['src/modules/blueprints/**/*.{ts,tsx}'],
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
    },
  },
  // Exception: Services layer allows underscore-prefixed unused vars (API signature compatibility)
  {
    files: ['src/services/**/*.{ts,tsx}'],
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],
    },
  },
  // Exception: test files may use hex colors (theme tests) and import internal
  // module paths directly (testing internals). Production code is still guarded
  // by the broader rules in the main block above.
  {
    files: ['**/*.test.{ts,tsx}', '**/__tests__/**'],
    rules: {
      'no-restricted-syntax': 'off',
      'import/no-internal-modules': 'off',
    },
  }
);
