import { defineConfig, devices } from '@playwright/test'

// Runs against the production images started by Docker Compose
// (`make test-e2e`), so the browser goes through Caddy, the Go API and
// PostgreSQL like a real user (tech spec §9).
export default defineConfig({
  testDir: './specs',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8088',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
