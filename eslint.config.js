// ESLint 配置文件（Flat Config 格式）
// 支持 JavaScript 和 TypeScript

import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';

export default [
  // JavaScript 文件配置
  {
    files: ["web/**/*.js", "scripts/**/*.js", "*.js"],
    languageOptions: {
      ecmaVersion: 2021,
      sourceType: "module",
      globals: {
        // 浏览器全局变量
        window: "readonly",
        document: "readonly",
        console: "readonly",
        navigator: "readonly",
        location: "readonly",
        localStorage: "readonly",
        sessionStorage: "readonly",
        setTimeout: "readonly",
        clearTimeout: "readonly",
        setInterval: "readonly",
        clearInterval: "readonly",
        fetch: "readonly",
        Promise: "readonly",
        URL: "readonly",
        URLSearchParams: "readonly",
        FormData: "readonly",
        confirm: "readonly",
        alert: "readonly",
        Blob: "readonly",
        FileReader: "readonly",
        WebSocket: "readonly",
        CustomEvent: "readonly",
        requestAnimationFrame: "readonly",
        CSS: "readonly",

        // Node.js 全局变量
        process: "readonly",
        __dirname: "readonly",
        __filename: "readonly",
        Buffer: "readonly",
        global: "readonly"
      }
    },
    rules: {
      // 变量和作用域
      "no-unused-vars": ["warn", {
        "args": "none",
        "varsIgnorePattern": "^_|^originalContent$|^isAutoBanned$|^isDisabled$"
      }],
      "no-undef": "error",
      "no-var": "error",
      "prefer-const": "warn",

      // 代码质量
      "no-console": "off",
      "no-debugger": "warn",
      "no-alert": "warn",
      "eqeqeq": ["error", "always", {"null": "ignore"}],
      "curly": ["error", "all"],
      "no-eval": "error",
      "no-implied-eval": "error",
      "no-with": "error",

      // 代码风格
      "semi": ["error", "always"],
      "quotes": ["warn", "single", {"avoidEscape": true}],
      "indent": ["warn", 2, {"SwitchCase": 1}],
      "comma-dangle": ["warn", "always-multiline"],
      "no-trailing-spaces": "warn",
      "eol-last": ["warn", "always"],

      // 最佳实践
      "no-duplicate-imports": "error",
      "no-useless-return": "warn",
      "prefer-arrow-callback": "warn",
      "prefer-template": "warn"
    }
  },

  // TypeScript 文件配置
  {
    files: ["web/**/*.ts", "web/**/*.tsx"],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 2021,
        sourceType: "module",
        project: "./web/tsconfig.json"
      }
    },
    plugins: {
      '@typescript-eslint': tseslint
    },
    rules: {
      // TypeScript 特定规则
      "@typescript-eslint/no-unused-vars": ["warn", {
        "args": "none",
        "varsIgnorePattern": "^_"
      }],
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/explicit-function-return-type": "off",
      "@typescript-eslint/explicit-module-boundary-types": "off",
      "@typescript-eslint/no-non-null-assertion": "warn",
      "@typescript-eslint/prefer-optional-chain": "warn",
      "@typescript-eslint/prefer-nullish-coalescing": "warn",

      // 禁用 JS 规则，使用 TS 版本
      "no-unused-vars": "off",

      // 代码质量
      "no-console": "off",
      "no-debugger": "warn",
      "eqeqeq": ["error", "always", {"null": "ignore"}],
      "curly": ["error", "all"],
      "no-eval": "error",

      // 代码风格
      "semi": ["error", "always"],
      "quotes": ["warn", "single", {"avoidEscape": true}],
      "indent": ["warn", 2, {"SwitchCase": 1}],
      "comma-dangle": ["warn", "always-multiline"],
      "no-trailing-spaces": "warn",
      "eol-last": ["warn", "always"]
    }
  },

  // 特定文件的全局变量
  {
    files: ["web/js/config.js", "web/js/creds.js", "web/js/oauth.js"],
    languageOptions: {
      globals: {
        ui: "readonly",
        escapeHTML: "readonly"
      }
    }
  },

  // 测试文件配置
  {
    files: ["**/*.test.ts", "**/*.test.js", "tests/**/*.ts", "tests/**/*.js"],
    languageOptions: {
      globals: {
        describe: "readonly",
        it: "readonly",
        expect: "readonly",
        beforeEach: "readonly",
        afterEach: "readonly",
        beforeAll: "readonly",
        afterAll: "readonly",
        vi: "readonly",
        test: "readonly"
      }
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "no-console": "off"
    }
  }
];
