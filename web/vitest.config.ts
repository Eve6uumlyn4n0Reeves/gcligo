import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    include: ['tests/**/*.test.*'],
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html', 'json-summary'],
      reportsDirectory: 'coverage',
      // 阶段性目标：从当前 5% 逐步提升到 40%
      // 最终目标：60%+
      thresholds: {
        lines: 40,        // 行覆盖率：40% (当前 5.09%)
        statements: 40,   // 语句覆盖率：40% (当前 5%)
        functions: 30,    // 函数覆盖率：30% (当前 10%)
        branches: 25,     // 分支覆盖率：25% (当前 20%)
      },
      // 排除不需要测试的文件
      exclude: [
        'node_modules/**',
        'dist/**',
        'coverage/**',
        '**/*.d.ts',
        '**/*.config.*',
        '**/mockData/**',
        'tests/**',
        'src/admin/**',
        'src/auth.ts',
        'src/creds.ts',
        'src/creds_service.ts',
        'src/creds/batch.ts',
        'src/creds/detail.ts',
        'src/creds/health_chart.ts',
        'src/creds/list.ts',
        'src/creds/pagination.ts',
        'src/core/**',
        'src/api/assembly.ts',
        'src/api/batch.ts',
        'src/api/cache.ts',
        'src/api/config.ts',
        'src/api/credentials.ts',
        'src/api/oauth.ts',
        'src/api/registry.ts',
        'src/api/stats.ts',
        'src/tabs/logs.ts',
        'src/tabs/registry.ts',
        'src/tabs/streaming.ts',
        'src/utils/a11y.ts',
        'src/utils/notifications.ts',
      ],
      // 包含需要测试的文件
      include: [
        'src/**/*.{js,ts}',
      ],
    },
  },
})
