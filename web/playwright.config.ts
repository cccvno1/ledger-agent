import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,        // 顺序执行，避免并发改数据互相影响
  retries: 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    // 无 AUTH_TOKEN 时免登录，直接测业务页面
    extraHTTPHeaders: {},
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // 测试前确认服务已起，不自动启动 dev server（使用正在运行的 Go 服务）
  webServer: undefined,
})
