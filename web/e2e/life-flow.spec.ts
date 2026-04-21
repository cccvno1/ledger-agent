import { test, expect, type Page } from '@playwright/test'

function parseMoney(text: string | null): number {
  if (!text) return 0
  const cleaned = text.replace(/[^\d.-]/g, '')
  return Number(cleaned || '0')
}

async function readSummaryValue(page: Page, label: string): Promise<number> {
  const text = await page
    .locator('.summary-item', { has: page.locator('.summary-label', { hasText: label }) })
    .locator('.summary-value')
    .first()
    .textContent()
  return parseMoney(text)
}

test('生活化场景：长期赊账客户分批还款后金额口径正确', async ({ page }) => {
  const customerName = `张三-${Date.now()}`

  await page.goto('/')
  await expect(page.locator('button', { hasText: '新增客户' })).toBeVisible({ timeout: 10000 })

  await page.locator('button', { hasText: '新增客户' }).click()
  await page.locator('.el-dialog input[placeholder="输入姓名"]').fill(customerName)
  await page.locator('.el-dialog button', { hasText: '保存' }).click()
  await expect(page.locator('.el-dialog')).not.toBeVisible({ timeout: 8000 })

  await page.locator('.search-input input').fill(customerName)
  const settledTitle = page.locator('.settled-title')
  if (await settledTitle.isVisible()) {
    const toggle = settledTitle.locator('button')
    if ((await toggle.textContent())?.includes('展开')) {
      await toggle.click()
    }
  }

  await page.locator('.customer-card', { hasText: customerName }).first().click()
  await expect(page).toHaveURL(/\/customers\//)

  await page.locator('button', { hasText: '记一笔账' }).click()
  await expect(page.locator('.el-drawer')).toBeVisible({ timeout: 5000 })

  // 第一笔：商品×数量×单价（出货单流程）
  const firstRow = page.locator('.el-drawer .table-row').nth(0)
  await firstRow.locator('.col-name input').fill('大米')
  await firstRow.locator('.col-qty input').fill('2')
  await firstRow.locator('.col-price input').fill('60')

  await page.locator('.el-drawer button', { hasText: '再加一行' }).click()
  const secondRow = page.locator('.el-drawer .table-row').nth(1)
  await secondRow.locator('.col-name input').fill('食用油')
  await secondRow.locator('.col-qty input').fill('1')
  await secondRow.locator('.col-price input').fill('60')

  await page.locator('.el-drawer button', { hasText: '记账' }).click()
  await expect(page.locator('.el-drawer')).not.toBeVisible({ timeout: 8000 })

  await expect.poll(async () => readSummaryValue(page, '累计出货')).toBe(180)
  await expect.poll(async () => readSummaryValue(page, '累计收款')).toBe(0)
  await expect.poll(async () => readSummaryValue(page, '当前欠款')).toBe(180)

  await page.getByRole('button', { name: /^收款$/ }).click()
  await expect(page.locator('.el-dialog', { hasText: '收款' })).toBeVisible({ timeout: 5000 })
  await page.locator('.el-dialog .el-input-number input').fill('80')
  await page.locator('.el-dialog button', { hasText: '确认收款' }).click()
  await expect(page.locator('.el-dialog')).not.toBeVisible({ timeout: 8000 })

  await expect.poll(async () => readSummaryValue(page, '累计收款')).toBe(80)
  await expect.poll(async () => readSummaryValue(page, '当前欠款')).toBe(100)

  await page.getByRole('button', { name: /^收款$/ }).click()
  await expect(page.locator('.el-dialog', { hasText: '收款' })).toBeVisible({ timeout: 5000 })
  await page.locator('.el-dialog button', { hasText: '全付' }).click()
  await page.locator('.el-dialog button', { hasText: '确认收款' }).click()
  await expect(page.locator('.el-dialog')).not.toBeVisible({ timeout: 8000 })

  await expect.poll(async () => readSummaryValue(page, '累计收款')).toBe(180)
  await expect.poll(async () => readSummaryValue(page, '当前欠款')).toBe(0)

  await page.locator('button', { hasText: '返回' }).click()
  await expect(page).toHaveURL('/')
})
