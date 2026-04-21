/**
 * E2E 测试 — 设置页 (SettingsView)
 *
 * 覆盖：
 *  1. 页面加载显示"微信通知"卡片
 *  2. 点击"获取绑定二维码"按钮后显示二维码图片
 *  3. 显示"请用微信扫码"提示
 *  4. 显示"等待扫码..."状态
 */
import { test, expect } from '@playwright/test'

test.describe('设置页', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings')
    await expect(page.locator('.settings-card')).toBeVisible({ timeout: 6000 })
  })

  test('显示微信通知卡片', async ({ page }) => {
    await expect(page.locator('button', { hasText: '返回首页' })).toBeVisible()
    await expect(page.locator('.card-title', { hasText: '微信通知' })).toBeVisible()
    await expect(page.locator('.card-desc')).toContainText('绑定微信后')
  })

  test('点击获取二维码后显示二维码图片', async ({ page }) => {
    const btn = page.locator('button', { hasText: /获取|重新获取/ })
    await btn.click()
    const img = page.locator('.qr-img')
    await expect(img).toBeVisible({ timeout: 8000 })
    const src = await img.getAttribute('src')
    expect(src).toBeTruthy()
    expect(src!).toContain('data:image/png;base64,')
    expect(src!.length).toBeGreaterThan(200)
  })

  test('二维码区域显示扫码提示', async ({ page }) => {
    const btn = page.locator('button', { hasText: /获取|重新获取/ })
    await btn.click()
    await expect(page.locator('.qr-hint')).toBeVisible({ timeout: 8000 })
    await expect(page.locator('.qr-hint')).toContainText('微信扫码')
  })

  test('显示等待扫码状态文字', async ({ page }) => {
    const btn = page.locator('button', { hasText: /获取|重新获取/ })
    await btn.click()
    await expect(page.locator('.qr-area')).toBeVisible({ timeout: 8000 })
    await expect(page.locator('.bind-pending')).toBeVisible({ timeout: 8000 })
    await expect(page.locator('.bind-pending')).toContainText('等待扫码')
  })
})
