import { defineConfig, devices } from '@playwright/test'

// E2E runs against the full stack (backend :8090 + MPC nodes + vite :5173).
// Start it first with ./start.sh, then: npm run e2e
export default defineConfig({
  testDir: './e2e',
  timeout: 120_000, // real MPC keygen + on-chain broadcast are slow
  expect: { timeout: 40_000 },
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: process.env.E2E_BASE ?? 'http://localhost:5173',
    headless: true,
    actionTimeout: 20_000,
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
